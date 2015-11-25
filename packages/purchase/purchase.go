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
)

type Purchase string

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var p *pkg.Pkg
var db *sqlx.DB
var ec *search.ElasticClient
var tc *twilio.Client
var sg *mail.Client

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
	// TODO allow the user to direct the conversation, e.g. say "something
	// more expensive" and have Ava respond appropriately

	log.Println("CURRENT STATE", getState())

	// if purchase has not been made, move user through the package's states
	err := updateState(m, resp, respMsg)
	if err != nil {
		return err
	}
	return pkg.SaveResponse(respMsg, resp)
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
		resp.Sentence = "Great! Where would you like it shipped?"
	case StateShippingAddress:
		// TODO add memory of shipping addresses
		addr, err := language.ExtractAddress(db, m.Input.Sentence)
		if err != nil {
			return err
		}
		if addr == nil {
			return nil
		}
		if err := m.User.SaveAddress(db, addr); err != nil {
			return err
		}
		if !statesShipping[addr.State] {
			resp.Sentence = "I'm sorry, but I can't legally ship wine to that state."
		}
		resp.State["shippingAddress"] = addr
		resp.State["state"] = StatePurchase
		price := getRecommendations()[getOffset()].Price
		// add shipping
		price += 1290 + uint64(len(getSelectedProducts())*100)
		// add tax
		tax := statesTax[addr.State]
		fullPrice := float64(price)
		if tax > 0.0 {
			fullPrice = price * (1.0 + tax)
		}
		taxInCents := int64(-1 * price)
		// ensure fractional cents are rounded up. technique stolen from
		// JavaScript. no need for math.CopySign() since it's unsigned
		price = uint64(fullPrice + 0.5)
		taxInCents = taxInCents + price
		resp.State["totalPrice"] = price
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
			getSelectedProducts(), getTotalPrice())
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
	for _, word := range words {
		switch word {
		case "detail", "details", "description", "more about", "review",
			"rating", "rated":
		case "price", "cost", "shipping", "how much":
		case "similar", "else", "different":
			resp.State["offset"] = getOffset() + 1
			if err := recommendProduct(resp, respMsg); err != nil {
				return err
			}
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

func getTotalPrice() uint64 {
	switch resp.State["totalPrice"].(type) {
	case uint64:
		return resp.State["totalPrice"].(uint64)
	case float64:
		return uint64(resp.State["totalPrice"].(float64))
	}
	return uint64(0)
}
