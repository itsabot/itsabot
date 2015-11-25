// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/shared/auth"
	"github.com/avabot/ava/shared/database"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/mail"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/search"
	"github.com/avabot/ava/shared/sms"
	"github.com/avabot/ava/shared/task"
)

type Purchase string

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var p *pkg.Pkg
var db *sqlx.DB
var ec *search.ElasticClient
var tc *twilio.Client
var sg *mail.Client
var tsk *task.Task

// resp enables the Run() function to skip to the FollowUp function if basic
// requirements are met.
var resp *dt.Resp

const (
	StateNone float64 = iota
	StatePreferences
	StateBudget
	StateRecommendations
	StateRecommendationsAlterBudget
	StateRecommendationsAlterQuery
	StateProductSelection
	StateShippingAddress
	StatePurchase
	StateComplete
)

var statesShipping = map[string]bool{
	"CA": true,
}

var statesTax = map[string]float64{
	"CA": 0.0925,
}

// TODO add support for purchasing multiple, upselling and promotions. Then a
// Task interface for follow ups and common multi-step information gathering,
// e.g. getting and naming addresses

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	var err error
	db, err = database.ConnectDB()
	if err != nil {
		log.Fatalln(err)
	}
	ec = search.NewClient()
	tc = sms.NewClient()
	sg = mail.NewClient()
	trigger := &dt.StructuredInput{
		Commands: language.Purchase(),
		Objects:  language.Alcohol(),
	}
	p, err = pkg.NewPackage("purchase", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	purchase := new(Purchase)
	if err := p.Register(purchase); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (t *Purchase) Run(m *dt.Msg, respMsg *dt.RespMsg) error {
	resp = m.NewResponse()
	resp.State = map[string]interface{}{
		"state":            StateNone,      // maintains state
		"query":            "",             // search query
		"budget":           "",             // suggested price
		"recommendations":  []dt.Product{}, // search results
		"offset":           uint(0),        // index in search
		"shippingAddress":  &dt.Address{},
		"productsSelected": []dt.Product{},
		"totalPrice":       uint64(0),
		"taxInCents":       uint64(0),
		"shippingInCents":  uint64(0),
	}
	si := m.Input.StructuredInput
	query := ""
	for _, o := range si.Objects {
		query += o + " "
	}
	// request longer query to get more interesting search results
	log.Println(query)
	if len(query) < 10 {
		resp.Sentence = "What do you look for in a wine?"
		resp.State["state"] = StatePreferences
		return pkg.SaveResponse(respMsg, resp)
	}
	// user provided us with a sufficiently detailed query, now search
	return t.FollowUp(m, respMsg)
}

func (t *Purchase) FollowUp(m *dt.Msg, respMsg *dt.RespMsg) error {
	if resp == nil {
		if err := m.GetLastResponse(db); err != nil {
			return err
		}
		resp = m.LastResponse
	}
	// have we already made the purchase?
	if getState() == StateComplete {
		// if so, reset state to allow for other purchases
		return t.Run(m, respMsg)
	}
	// allow the user to direct the conversation, e.g. say "something more
	// expensive" and have Ava respond appropriately
	if getState() >= StateRecommendations {
		if err := handleKeywords(m, resp, respMsg); err != nil {
			return err
		}
		return pkg.SaveResponse(respMsg, resp)
	}
	// if purchase has not been made, move user through the package's states
	if err := updateState(m, resp, respMsg); err != nil {
		return err
	}
	if tsk == nil {
		return pkg.SaveResponse(respMsg, resp)
	}
	return nil
}

func updateState(m *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) error {
	switch getState() {
	case StatePreferences:
		// TODO ensure Ava remembers past answers for preferences
		// "I know you're going to love this"
		resp.State["query"] = getQuery() + " " + m.Input.Sentence
		resp.State["state"] = StateBudget
		resp.Sentence = "Ok. How much do you usually pay for a bottle of wine?"
	case StateBudget:
		// TODO ensure Ava remembers past answers for budget
		val, budget, err := language.ExtractCurrency(m.Input.Sentence)
		if err != nil {
			log.Println("err extracting currency")
			return err
		}
		if budget == nil {
			log.Println("no budget found")
			return nil
		}
		log.Println("set budget", val)
		resp.State["budget"] = val
		resp.State["state"] = StateRecommendations
		fallthrough
	case StateRecommendations:
		err := setRecs(resp, respMsg)
		if err != nil {
			return err
		}
		if err = recommendProduct(resp, respMsg); err != nil {
			return err
		}
	case StateRecommendationsAlterBudget:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return nil
		}
		if yes.Bool {
			resp.State["budget"] = uint64(15000)
		} else {
			resp.State["query"] = "wine"
			if getBudget() < 1500 {
				resp.State["budget"] = uint64(1500)
			}
		}
		resp.State["offset"] = 0
		resp.State["state"] = StateRecommendations
		return updateState(m, resp, respMsg)
	case StateRecommendationsAlterQuery:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return handleKeywords(m, resp, respMsg)
		}
		if yes.Bool {
			resp.State["query"] = "wine"
			if getBudget() < 1500 {
				resp.State["budget"] = uint64(1500)
			}
		} else {
			resp.Sentence = "Ok. Let me know if there's anything else with which I can help you."
		}
		resp.State["offset"] = 0
		resp.State["state"] = StateRecommendations
		return updateState(m, resp, respMsg)
	case StateProductSelection:
		// was the recommendation Ava made good?
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			log.Println("HERE")
			resp.Sentence = "I'm not sure I understand you. Should we order the wine?"
			return handleKeywords(m, resp, respMsg)
		}
		if !yes.Bool {
			resp.State["offset"] = getOffset() + 1
			return nil
		}
		selection, err := currentSelection(resp.State)
		if err != nil {
			return err
		}
		resp.State["productsSelected"] = append(getSelectedProducts(),
			*selection)
		resp.State["state"] = StateShippingAddress
		fallthrough
	case StateShippingAddress:
		// tasks are multi-step processes often useful for several
		// packages
		var err error
		var addr *dt.Address
		tsk, err = task.New(db, u, resp, respMsg, pkg.PkgConfig.Name)
		done, err := tsk.RequestAddress(addr)
		if err != nil {
			return err
		}
		if !done {
			return nil
		}
		// addr is now guaranteed to be populated
		if !statesShipping[addr.State] {
			resp.Sentence = "I'm sorry, but I can't legally ship wine to that state."
		}
		resp.State["shippingAddress"] = addr
		resp.State["state"] = StatePurchase
		price := getRecommendations()[getOffset()].Price
		// calculate shipping. note that this is vendor specific
		shippingInCents := 1290 + uint64((len(getSelectedProducts())-1)*120)
		price += shippingInCents
		// add tax
		tax := statesTax[addr.State]
		if tax > 0.0 {
			tax *= price
		}
		// ensure fractional cents are rounded up. technique stolen from
		// JavaScript. no need for math.CopySign() since it's unsigned
		taxInCents := uint64(tax + 0.5)
		price += taxInCents
		resp.State["totalPrice"] = price
		resp.State["taxInCents"] = taxInCents
		resp.State["shippingInCents"] = shippingInCents
		tmp := fmt.Sprintf("$%.2f including shipping and tax. ",
			float64(price)/100)
		tmp += "Should I place the order?"
		resp.Sentence = fmt.Sprintf("Ok. It comes to %s", tmp)
	case StatePurchase:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return nil
		}
		if !yes.Bool {
			return handleKeywords(m, resp, respMsg)
		}
		// TODO
		// ensure Ava follows up to ensure the delivery occured, get
		// feedback, etc.
		err := auth.Purchase(db, tc, sg, auth.MethodZip, m,
			getSelectedProducts(), getPrices())
		if err != nil {
			return err
		}
		resp.State["state"] = StateComplete
		resp.Sentence = "Great! I've placed the order. You'll receive a confirmation by email."
	}
	return nil
}

func currentSelection(state map[string]interface{}) (*dt.Product, error) {
	recs := getRecommendations()
	l := uint(len(recs))
	if l == 0 {
		return &dt.Product{}, errors.New("empty recommendations")
	}
	offset := getOffset()
	if l <= offset {
		err := errors.New("offset exceeds recommendation length")
		return &dt.Product{}, err
	}
	return &recs[offset], nil
}

func handleKeywords(m *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) error {
	words := strings.Fields(m.Input.Sentence)
	modifier := 1
	for _, word := range words {
		switch word {
		case "detail", "details", "description", "more about", "review",
			"rating", "rated":
			resp.Sentence = "Every wine I recommend is at the top of its craft."
			return nil
		case "price", "cost", "shipping", "how much", "total":
			prices := getPrices()
			itemCost := prices[0] - prices[1] - prices[2]
			s := fmt.Sprintf("The items cost %.2f, ", itemCost)
			s += fmt.Sprintf("shipping is %.2f and ", prices[1])
			if prices[2] == 0.0 {
				s += fmt.Sprintf("tax is %.2f, ", prices[2])
			}
			s += fmt.Sprintf("totaling %.2f.", prices[0])
			resp.Sentence = s
		case "similar", "else", "different":
			resp.State["offset"] = getOffset() + 1
			if err := recommendProduct(resp, respMsg); err != nil {
				return err
			}
		case "expensive", "event", "nice", "nicer":
			// perfect example of a need for stemming
			budg := getBudget()
			if budg >= 10000 {
				resp.State["budget"] = budg + (10000 * modifier)
			} else if budg >= 5000 {
				resp.State["budget"] = budg + (5000 * modifier)
			} else {
				resp.State["budget"] = budg + (2500 * modifier)
			}
		case "more", "special":
			modifier = 1
		case "less":
			modifier = -1
		}
	}
	return nil
}

func recommendProduct(resp *dt.Resp, respMsg *dt.RespMsg) error {
	recs := getRecommendations()
	if len(recs) == 0 {
		words := strings.Fields(getQuery())
		if len(words) == 1 {
			resp.Sentence = "I couldn't find any wines like that. "
			if getBudget() < 5000 {
				resp.Sentence += "Should we look among the more expensive bottles?"
				resp.State["state"] = StateRecommendationsAlterBudget
			} else {
				resp.Sentence += "Should we expand your search to more wines?"
				resp.State["state"] = StateRecommendationsAlterQuery
			}
			return nil
		} else {
			resp.State["query"] = "simple"
			return nil
		}
	}
	log.Println("showing product")
	product := recs[getOffset()]
	tmp := fmt.Sprintf("A %s for $%.2f. ", product.Name,
		float64(product.Price)/100)
	if len(product.Reviews) > 0 {
		summary, err := language.Summarize(
			product.Reviews[0].Body, "products_alcohol")
		if err != nil {
			return err
		}
		if len(summary) > 0 {
			tmp += summary + " "
		}
	}
	tmp += "Does that sound good?"
	resp.Sentence = language.SuggestedProduct(tmp)
	resp.State["state"] = StateProductSelection
	return nil
}

func setRecs(resp *dt.Resp, respMsg *dt.RespMsg) error {
	results, err := ec.FindProducts(getQuery(), "alcohol", getBudget(), 20)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		resp.Sentence = "I'm sorry. I couldn't find anything like that."
	}
	// TODO - better recommendations
	// results = sales.SortByRecommendation(results)
	resp.State["recommendations"] = results
	return nil
}

// TODO customize the type of resp.State, forcing all reads and writes through
// these getter/setter functions to preserve and handle types across interface{}
func getOffset() uint {
	switch resp.State["offset"].(type) {
	case uint:
		return resp.State["offset"].(uint)
	case float64:
		return uint(resp.State["offset"].(float64))
	default:
		log.Println("warn: couldn't get offset: invalid type",
			reflect.TypeOf(resp.State["offset"]))
	}
	return uint(0)
}

func getQuery() string {
	return resp.State["query"].(string)
}

func getBudget() uint64 {
	switch resp.State["budget"].(type) {
	case uint64:
		return resp.State["budget"].(uint64)
	case float64:
		return uint64(resp.State["budget"].(float64))
	default:
		log.Println("warn: couldn't get budget: invalid type",
			reflect.TypeOf(resp.State["budget"]))
	}
	return uint64(0)
}

func getSelectedProducts() []dt.Product {
	products, ok := resp.State["productsSelected"].([]dt.Product)
	if !ok {
		return nil
	}
	return products
}

func getRecommendations() []dt.Product {
	products, ok := resp.State["recommendations"].([]dt.Product)
	if !ok {
		return nil
	}
	return products
}

func getState() float64 {
	return resp.State["state"].(float64)
}

func getPrices() [3]uint64 {
	keys := [3]string{"totalPrice", "taxInCents", "shippingInCents"}
	vals := [3]uint64{}
	for _, key := range keys {
		switch resp.State[key].(type) {
		case uint64:
			vals = append(vals, resp.State[key].(uint64))
		case float64:
			vals = append(uint64(resp.State[key].(float64)))
		default:
			typ := reflect.TypeOf(resp.State[key])
			log.Printf("warn: invalid type %s for %s\n", typ, key)
		}
	}
	return vals
}
