// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/renstrom/fuzzysearch/fuzzy"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/prefs"
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
	StateSetRecommendations
	StateRecommendationsAlterBudget
	StateRecommendationsAlterQuery
	StateMakeRecommendation
	StateProductSelection
	StateContinueShopping
	StateShippingAddress
	StatePurchase
	StateAuth
	StateComplete
)

const pkgName string = "purchase"

var statesShipping = map[string]bool{
	"CA": true,
}

var statesTax = map[string]float64{
	"CA": 0.0925,
}

// TODO add support for upselling and promotions. Then a Task interface for
// follow ups

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
	p, err = pkg.NewPackage(pkgName, *port, trigger)
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
	tastePref, err := prefs.Get(ctx.DB, resp.UserID, pkgName,
		prefs.KeyTaste)
	if err != nil {
		return err
	}
	if len(tastePref) == 0 {
		resp.State["query"] = query + " " + tastePref
		resp.State["state"] = StatePreferences
		resp.Sentence = "Sure. What do you usually look for in a wine?"
		return p.SaveResponse(respMsg, resp)
	}
	resp.State["query"] = tastePref
	budgetPref, err := prefs.Get(ctx.DB, resp.UserID, pkgName,
		prefs.KeyBudget)
	if err != nil {
		return err
	}
	if len(budgetPref) > 0 {
		resp.State["budget"], err = strconv.ParseUint(budgetPref, 10,
			64)
		if err != nil {
			return err
		}
		resp.State["state"] = StateSetRecommendations
		updateState(m, resp, respMsg)
		return p.SaveResponse(respMsg, resp)
	}
	resp.State["state"] = StateBudget
	resp.Sentence = "Sure. How much do you usually pay for a bottle?"
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
	var kw bool
	if getState() > StateSetRecommendations {
		log.Println("handling keywords")
		var err error
		kw, err = handleKeywords(m, resp, respMsg)
		if err != nil {
			return err
		}
	}
	if !kw {
		// if purchase has not been made, move user through the
		// package's states
		log.Println("updating state", getState())
		if err := updateState(m, resp, respMsg); err != nil {
			return err
		}
	}
	return p.SaveResponse(respMsg, resp)
}

func updateState(m *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) error {
	switch getState() {
	case StatePreferences:
		// TODO ensure Ava remembers past answers for preferences
		// "I know you're going to love this"
		if getBudget() == 0 {
			resp.State["query"] = getQuery() + " " + m.Input.Sentence
			resp.State["state"] = StateBudget
			resp.Sentence = "Ok. How much do you usually pay for a bottle of wine?"
			if err := prefs.Save(ctx.DB, resp.UserID, pkgName,
				prefs.KeyTaste, getQuery()); err != nil {
				log.Println("err: saving budget pref")
				return err
			}
		} else {
			resp.State["state"] = StateSetRecommendations
			return updateState(m, resp, respMsg)
		}
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
		resp.State["budget"] = val
		resp.State["state"] = StateSetRecommendations
		err = prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyBudget,
			strconv.FormatUint(getBudget(), 10))
		if err != nil {
			log.Println("err: saving budget pref")
			return err
		}
		fallthrough
	case StateSetRecommendations:
		log.Println("setting recs")
		err := setRecs(resp, respMsg)
		if err != nil {
			log.Println("err setting recs")
			return err
		}
		resp.State["state"] = StateMakeRecommendation
		return updateState(m, resp, respMsg)
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
		resp.State["state"] = StateSetRecommendations
		err := prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyBudget,
			strconv.FormatUint(getBudget(), 10))
		if err != nil {
			return err
		}
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
		resp.State["state"] = StateSetRecommendations
		err := prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyBudget,
			strconv.FormatUint(getBudget(), 10))
		if err != nil {
			return err
		}
		return updateState(m, resp, respMsg)
	case StateMakeRecommendation:
		log.Println("recommending product")
		if err := recommendProduct(resp, respMsg); err != nil {
			return err
		}
	case StateProductSelection:
		// was the recommendation Ava made good?
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			resp.Sentence = "I'm not sure I understand you. Should we order the wine?"
			return nil
		}
		if !yes.Bool {
			resp.State["offset"] = getOffset() + 1
			log.Println("updating offset", getOffset())
			resp.State["state"] = StateMakeRecommendation
			return updateState(m, resp, respMsg)
		}
		selection, err := currentSelection(resp.State)
		if err != nil {
			return err
		}
		resp.Sentence = "Ok, I've added it to your cart. Should we look for a few more?"
		resp.State["productsSelected"] = append(getSelectedProducts(),
			*selection)
		resp.State["state"] = StateContinueShopping
	case StateContinueShopping:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			resp.Sentence = "I'm not sure I understand you. Should we look for more wines? At any time, let me know if you want to see your cart or checkout."
			return nil
		}
		if yes.Bool {
			resp.State["offset"] = getOffset() + 1
			if err := recommendProduct(resp, respMsg); err != nil {
				return err
			}
			resp.State["state"] = StateProductSelection
			return nil
		}
		resp.State["state"] = StateShippingAddress
		return updateState(m, resp, respMsg)
	case StateShippingAddress:
		prods := getSelectedProducts()
		if len(prods) == 0 {
			resp.Sentence = "You haven't picked any products. Should we keep looking?"
			resp.State["state"] = StateContinueShopping
			return nil
		}
		// tasks are multi-step processes often useful across several
		// packages
		var addr *dt.Address
		tsk, err := task.New(ctx, resp, respMsg)
		if err != nil {
			return err
		}
		done, err := tsk.RequestAddress(&addr, len(prods))
		if err != nil {
			return err
		}
		if !done {
			return nil
		}
		if !statesShipping[addr.State] {
			resp.Sentence = "I'm sorry, but I can't legally ship wine to that state."
		}
		resp.State["shippingAddress"] = addr
		price := uint64(0)
		for _, prod := range prods {
			price += prod.Price
		}
		// calculate shipping. note that this is vendor specific
		shippingInCents := 1290 + uint64((len(prods)-1)*120)
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
		resp.State["state"] = StatePurchase
	case StatePurchase:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return nil
		}
		if !yes.Bool {
			resp.Sentence = "Ok."
			return nil
		}
		resp.State["state"] = StateAuth
		return updateState(m, resp, respMsg)
	case StateAuth:
		// TODO ensure Ava follows up to ensure the delivery occured,
		// get feedback, etc.
		prods := getSelectedProducts()
		purchase, err := dt.NewPurchase(ctx, &dt.PurchaseConfig{
			User:            m.User,
			ShippingAddress: getShippingAddress(),
			VendorID:        prods[0].VendorID,
			Prices:          getPrices(),
			Products:        prods,
		})
		if err != nil {
			return err
		}
		tsk, err := task.New(ctx, resp, respMsg)
		if err != nil {
			return err
		}
		log.Println("task init")
		done, err := tsk.RequestPurchase(task.MethodZip,
			getSelectedProducts(), purchase)
		log.Println("task fired. request purchase")
		if err == task.ErrInvalidAuth {
			resp.Sentence = "I'm sorry but that doesn't match what I have. You could try to add a new card here: https://avabot.com/?/cards/new"
			return nil
		}
		if err != nil {
			log.Println("err requesting purchase")
			return err
		}
		if !done {
			log.Println("purchase incomplete")
			return nil
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
		log.Println("!!! empty recs !!!")
		log.Println("query", getQuery())
		log.Println("offset", getOffset())
		log.Println("budget", getBudget())
		log.Println("selectedProducts", len(getSelectedProducts()))
		return &dt.Product{}, errors.New("empty recommendations")
	}
	offset := getOffset()
	if l <= offset {
		err := errors.New("offset exceeds recommendation length")
		return &dt.Product{}, err
	}
	return &recs[offset], nil
}

func handleKeywords(m *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) (bool,
	error) {
	words := strings.Fields(m.Input.Sentence)
	modifier := 1
	kwMatch := false
	for _, word := range words {
		switch word {
		case "detail", "details", "description", "more about", "review",
			"rating", "rated":
			r := rand.Intn(3)
			switch r {
			case 0:
				resp.Sentence = "Every wine I recommend is at the top of its craft."
			case 1:
				resp.Sentence = "I only recommend the best."
			case 2:
				resp.Sentence = "This wine has been personally selected by leading wine experts."
			}
			kwMatch = true
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
			kwMatch = true
		case "look":
			log.Println("HERE")
			resp.State["offset"] = 0
			resp.State["query"] = m.Input.Sentence
			resp.State["state"] = StateMakeRecommendation
			if err := recommendProduct(resp, respMsg); err != nil {
				return true, err
			}
			kwMatch = true
		case "similar", "else", "different":
			resp.State["offset"] = getOffset() + 1
			if err := recommendProduct(resp, respMsg); err != nil {
				return true, err
			}
			kwMatch = true
		case "expensive", "event", "nice", "nicer", "cheap", "cheaper":
			// perfect example of a need for stemming
			if word == "cheap" || word == "cheaper" {
				modifier = -1
			}
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
			kwMatch = true
		case "more", "special":
			modifier = 1
		case "less":
			modifier = -1
		case "cart":
			kwMatch = true
			prods := getSelectedProducts()
			var prodNames []string
			for _, prod := range prods {
				prodNames = append(prodNames, prod.Name)
			}
			if len(prods) == 0 {
				resp.Sentence = "You haven't picked any wines, yet."
			} else if len(prods) == 1 {
				resp.Sentence = "You've picked a " +
					prodNames[0] + "."
			} else {
				resp.Sentence = fmt.Sprintf(
					"You've picked %d wines: ", len(prods))
				resp.Sentence += language.SliceToString(
					prodNames, "and") + "."
			}
			r := rand.Intn(2)
			switch r {
			case 0:
				resp.Sentence += " Should we keep looking or checkout?"
			case 1:
				resp.Sentence += " Should we add some more or checkout?"
			}
			resp.State["state"] = StateContinueShopping
		case "checkout", "check":
			kwMatch = false // deliberately allow pass to updateState
			resp.State["state"] = StateShippingAddress
		case "remove", "rid", "drop":
			kwMatch = true
			prods := getSelectedProducts()
			var prodNames []string
			for _, prod := range prods {
				prodNames = append(prodNames, prod.Name)
			}
			p := fuzzy.Find(m.Input.Sentence, prodNames)
			if len(p) == 0 {
				resp.Sentence = "I couldn't find a wine like that in your cart. "
			} else {
				resp.Sentence = fmt.Sprintf(
					"Ok, I'll remove %s.", p[0])
			}
			r := rand.Intn(2)
			switch r {
			case 0:
				resp.Sentence += " Is there something else I can help you find?"
			case 1:
				resp.Sentence += " Would you like to find another?"
			}
			resp.State["state"] = StateContinueShopping
		}
	}
	return kwMatch, nil
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
	r := rand.Intn(2)
	switch r {
	case 0:
		tmp += "Does that sound good?"
	case 1:
		tmp += "Should I add it to your cart?"
	}
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
	case int:
		return uint(resp.State["offset"].(int))
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
		prodMap, ok := resp.State["productsSelected"].(interface{})
		if !ok {
			log.Println("productsSelected not found",
				resp.State["productsSelected"])
			return nil
		}
		byt, err := json.Marshal(prodMap)
		if err != nil {
			log.Println("err: marshaling products", err)
		}
		if err = json.Unmarshal(byt, &products); err != nil {
			log.Println("err: unmarshaling products", err)
		}
	}
	return products
}

func getRecommendations() []dt.Product {
	products, ok := resp.State["recommendations"].([]dt.Product)
	if !ok {
		prodMap, ok := resp.State["recommendations"].(interface{})
		if !ok {
			log.Println("recommendations not found",
				resp.State["recommendations"])
			return nil
		}
		byt, err := json.Marshal(prodMap)
		if err != nil {
			log.Println("err: marshaling products", err)
		}
		if err = json.Unmarshal(byt, &products); err != nil {
			log.Println("err: unmarshaling products", err)
		}
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
