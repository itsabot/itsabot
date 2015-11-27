// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/avabot/ava/shared/auth"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/task"
)

type Purchase string

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var p *pkg.Pkg
var ctx *dt.Ctx

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
	ctx, err = dt.NewContext()
	if err != nil {
		log.Fatalln(err)
	}
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
	ctx.Msg = m
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
	if len(query) < 10 {
		resp.Sentence = "What do you look for in a wine?"
		resp.State["state"] = StatePreferences
		return pkg.SaveResponse(respMsg, resp)
	}
	// user provided us with a sufficiently detailed query, now search
	return t.FollowUp(m, respMsg)
}

func (t *Purchase) FollowUp(m *dt.Msg, respMsg *dt.RespMsg) error {
	ctx.Msg = m
	if resp == nil {
		if err := m.GetLastResponse(ctx.DB); err != nil {
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
	}
	// if purchase has not been made, move user through the package's states
	if err := updateState(m, resp, respMsg); err != nil {
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
			return nil
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
			resp.Sentence = "I'm not sure I understand you. Should we order the wine?"
			return nil
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
		// tasks are multi-step processes often useful across several
		// packages
		var addr *dt.Address
		tsk, err := task.New(ctx.DB, m, resp, respMsg)
		if err != nil {
			return err
		}
		done, err := tsk.RequestAddress(&addr)
		if err != nil {
			return err
		}
		if !done {
			return nil
		}
		log.Printf("HERE: %+v\n")
		if !statesShipping[addr.State] {
			resp.Sentence = "I'm sorry, but I can't legally ship wine to that state."
		}
		log.Println("HERE 1")
		resp.State["shippingAddress"] = addr
		resp.State["state"] = StatePurchase
		selection, err := currentSelection(resp.State)
		if err != nil {
			return err
		}
		price := selection.Price
		// calculate shipping. note that this is vendor specific
		shippingInCents := 1290 + uint64((len(getSelectedProducts())-1)*120)
		// add tax
		tax := statesTax[addr.State]
		if tax > 0.0 {
			tax *= float64(price)
		}
		// ensure fractional cents are rounded up. technique stolen from
		// JavaScript. no need for math.CopySign() since it's unsigned
		taxInCents := uint64(tax + 0.5)
		price += taxInCents
		price += shippingInCents
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
			return nil
		}
		// TODO
		// ensure Ava follows up to ensure the delivery occured, get
		// feedback, etc.
		purchase := dt.NewPurchase(ctx.DB)
		purchase.UserID = m.User.ID
		purchase.User = m.User
		purchase.VendorID = getSelectedProducts()[0].VendorID
		purchase.ShippingAddress = getShippingAddress()
		purchase.ShippingAddressID = sql.NullInt64{
			Int64: int64(purchase.ShippingAddress.ID),
			Valid: true,
		}
		for _, p := range getSelectedProducts() {
			purchase.Products = append(purchase.Products, p.Name)
		}
		prices := getPrices()
		purchase.Tax = prices[1]
		purchase.Shipping = prices[2]
		purchase.Total = prices[0]
		purchase.AvaFee = uint64(float64(prices[0]) * 0.05 * 100)
		purchase.CreditCardFee = uint64(
			(float64(prices[0])*0.029 + 0.3) * 100)
		purchase.TransferFee =
			uint64((float64(purchase.Total-
				purchase.AvaFee-
				purchase.CreditCardFee) * 0.005) * 100)
		purchase.VendorPayout = purchase.Total -
			purchase.AvaFee -
			purchase.CreditCardFee -
			purchase.TransferFee
		t := time.Now().Add(7 * 24 * time.Hour)
		purchase.DeliveryExpectedAt = &t
		if err := purchase.Init(); err != nil {
			return err
		}
		err := auth.Purchase(ctx, auth.MethodZip, getSelectedProducts(),
			purchase)
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
			var tmp int
			if budg >= 10000 {
				tmp = int(budg) + (10000 * modifier)
			} else if budg >= 5000 {
				tmp = int(budg) + (5000 * modifier)
			} else {
				tmp = int(budg) + (2500 * modifier)
			}
			if tmp <= 0 {
				tmp = 0
			}
			resp.State["budget"] = uint64(tmp)
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
	results, err := ctx.EC.FindProducts(getQuery(), "alcohol", getBudget(),
		20)
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

func getShippingAddress() *dt.Address {
	addr, ok := resp.State["shippingAddress"].(*dt.Address)
	if !ok {
		return nil
	}
	return addr
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

// getPrices requires a slice of length 3, as it's used directly in
// auth.Purchase
func getPrices() []uint64 {
	keys := []string{"totalPrice", "taxInCents", "shippingInCents"}
	vals := []uint64{}
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		switch resp.State[key].(type) {
		case uint64:
			vals = append(vals, resp.State[key].(uint64))
		case float64:
			vals = append(vals, uint64(resp.State[key].(float64)))
		default:
			typ := reflect.TypeOf(resp.State[key])
			log.Printf("warn: invalid type %s for %s\n", typ, key)
		}
	}
	return vals
}
