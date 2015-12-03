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

var ErrEmptyRecommendations = errors.New("empty recommendations")
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
		"state":            StateNone,        // maintains state
		"query":            "",               // search query
		"budget":           "",               // suggested price
		"recommendations":  dt.ProductSels{}, // search results
		"offset":           uint(0),          // index in search
		"shippingAddress":  &dt.Address{},
		"productsSelected": dt.ProductSels{},
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
		resp.Sentence = "Sure. What do you usually look for in a wine? (e.g. dry, fruity, sweet, earthy, oak, etc.)"
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
		resp.Sentence = ""
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
	log.Println("state", getState())
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
		val, err := language.ExtractCurrency(m.Input.Sentence)
		if err != nil {
			log.Println("err extracting currency")
			return err
		}
		if !val.Valid {
			return nil
		}
		resp.State["budget"] = val.Int64
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
			log.Println("StateProductSelection: yes invalid")
			return nil
		}
		if !yes.Bool {
			resp.State["offset"] = getOffset() + 1
			log.Println("updating offset", getOffset())
			resp.State["state"] = StateMakeRecommendation
			return updateState(m, resp, respMsg)
		}
		log.Println("StateProductSelection: yes valid and true")
		count := language.ExtractCount(m.Input.Sentence)
		if count.Valid {
			if count.Int64 == 0 {
				// asked to order 0 wines. trigger confused
				// reply
				return nil
			}
		}
		selection, err := currentSelection(resp.State)
		if err == ErrEmptyRecommendations {
			resp.Sentence = "I couldn't find any wines like that. "
			if getBudget() < 5000 {
				resp.Sentence += "Should we look among the more expensive bottles?"
				resp.State["state"] = StateRecommendationsAlterBudget
			} else {
				resp.Sentence += "Should we expand your search to more wines?"
				resp.State["state"] = StateRecommendationsAlterQuery
			}
			return updateState(m, resp, respMsg)
		}
		if err != nil {
			return err
		}
		if !count.Valid || count.Int64 <= 1 {
			count.Int64 = 1
			resp.Sentence = "Ok, I've added it to your cart. Should we look for a few more?"
		} else if uint(count.Int64) > selection.Stock {
			resp.Sentence = "I'm sorry, but I don't have that many available. Should we do "
			return nil
		} else {
			resp.Sentence = fmt.Sprintf(
				"Ok, I'll add %d to your cart. Should we look for a few more?",
				count.Int64)
		}
		prod := dt.ProductSel{
			Product: selection,
			Count:   uint(count.Int64),
		}
		resp.State["productsSelected"] = append(getSelectedProducts(),
			prod)
		resp.State["state"] = StateContinueShopping
	case StateContinueShopping:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return nil
		}
		if yes.Bool {
			resp.State["offset"] = getOffset() + 1
			resp.State["state"] = StateMakeRecommendation
		} else {
			resp.State["state"] = StateShippingAddress
		}
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
		if addr == nil {
			return errors.New("addr is nil")
		}
		if !statesShipping[addr.State] {
			resp.Sentence = "I'm sorry, but I can't legally ship wine to that state."
		}
		resp.State["shippingAddress"] = addr
		tmp := fmt.Sprintf("$%.2f including shipping and tax. ",
			float64(prods.Prices(addr)["total"])/100)
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
			ProductSels:     prods,
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
		return nil, ErrEmptyRecommendations
	}
	offset := getOffset()
	if l <= offset {
		err := errors.New("offset exceeds recommendation length")
		return nil, err
	}
	return &recs[offset], nil
}

func handleKeywords(m *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) (bool,
	error) {
	words := strings.Fields(strings.ToLower(m.Input.Sentence))
	modifier := 1
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
		case "price", "cost", "shipping", "total":
			prices := getSelectedProducts().
				Prices(getShippingAddress())
			s := fmt.Sprintf("The items cost $%.2f, ",
				float64(prices["products"])/100)
			s += fmt.Sprintf("shipping is $%.2f, ",
				float64(prices["shipping"])/100)
			if prices["tax"] > 0.0 {
				s += fmt.Sprintf("and tax is $%.2f, ",
					float64(prices["tax"])/100)
			}
			s += fmt.Sprintf("totaling $%.2f.",
				float64(prices["total"])/100)
			resp.Sentence = s
		case "find", "search", "show":
			resp.State["offset"] = 0
			resp.State["query"] = m.Input.Sentence
			resp.State["state"] = StateSetRecommendations
			err := prefs.Save(ctx.DB, ctx.Msg.User.ID, pkgName,
				prefs.KeyTaste, m.Input.Sentence)
			if err != nil {
				return false, err
			}
		case "similar", "else", "different", "looking", "look":
			resp.State["offset"] = getOffset() + 1
			resp.State["state"] = StateMakeRecommendation
		case "expensive", "event", "nice", "nicer", "cheap", "cheaper":
			// perfect example of a need for stemming
			if word == "cheap" || word == "cheaper" {
				modifier = -1
			}
			budg := getBudget()
			var tmp int
			if budg >= 10000 {
				tmp = int(budg) + (5000 * modifier)
			} else if budg >= 5000 {
				tmp = int(budg) + (2500 * modifier)
			} else {
				tmp = int(budg) + (1500 * modifier)
			}
			if tmp <= 0 {
				tmp = 1000
			}
			resp.State["budget"] = uint64(tmp)
			resp.State["state"] = StateSetRecommendations
			err := prefs.Save(ctx.DB, ctx.Msg.User.ID, pkgName,
				prefs.KeyBudget, strconv.Itoa(tmp))
			if err != nil {
				return false, err
			}
		case "cart":
			prods := getSelectedProducts()
			var prodNames []string
			for _, prod := range prods {
				name := fmt.Sprintf("%s (%dx$%.2f)", prod.Name,
					prod.Count, float64(prod.Price)/100)
				prodNames = append(prodNames, name)
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
			if len(prods) > 0 {
				var tmp string
				r := rand.Intn(2)
				switch r {
				case 0:
					tmp = " Should we checkout?"
				case 1:
					tmp = " Are you ready to checkout?"
				}
				// 255 is the database varchar limit, but we should aim
				// to be below 140 (sms)
				if len(resp.Sentence) > 140-len(tmp) {
					// 4 refers to the length of the ellipsis
					resp.Sentence = resp.Sentence[0 : 140-len(tmp)-4]
					resp.Sentence += "... "
				}
				resp.Sentence += tmp
			}
			resp.State["state"] = StateContinueShopping
		case "checkout", "check", "done":
			prods := getSelectedProducts()
			if len(prods) == 1 {
				tmp := fmt.Sprintf(
					"Ok. Where should I ship your bottle of %s?",
					prods[0].Name)
				resp.Sentence = tmp
			} else if len(prods) > 1 {
				resp.Sentence = fmt.Sprintf(
					"Ok. Where should I ship these %d bottles?",
					len(prods))
			}
			resp.State["state"] = StateShippingAddress
		case "remove", "rid", "drop":
			prods := getSelectedProducts()
			var prodNames []string
			for _, prod := range prods {
				prodNames = append(prodNames, prod.Name)
			}
			var matches []string
			for _, w := range strings.Fields(m.Input.Sentence) {
				if len(w) <= 3 {
					continue
				}
				tmp := fuzzy.FindFold(w, prodNames)
				if len(tmp) > 0 {
					matches = append(matches, tmp...)
				}
			}
			if len(matches) == 0 {
				resp.Sentence = "I couldn't find a wine like that in your cart."
			} else if len(matches) == 1 {
				resp.Sentence = fmt.Sprintf(
					"Ok, I'll remove %s.", matches[0])
				removeSelectedProduct(matches[0])
			} else {
				resp.Sentence = "Ok, I'll remove those."
				for _, match := range matches {
					removeSelectedProduct(match)
				}
			}
			r := rand.Intn(2)
			switch r {
			case 0:
				resp.Sentence += " Is there something else I can help you find?"
			case 1:
				resp.Sentence += " Would you like to find another?"
			}
			resp.State["state"] = StateContinueShopping
		case "help", "command":
			resp.Sentence = "At any time you can ask to see your cart, checkout, find something different (dry, fruity, earthy, etc.), or find something more or less expensive."
		case "more", "special":
			modifier *= modifier
		case "less":
			modifier *= modifier
		case "much", "very", "extremely":
			modifier *= 2
		}
	}
	return len(resp.Sentence) > 0, nil
}

func recommendProduct(resp *dt.Resp, respMsg *dt.RespMsg) error {
	recs := getRecommendations()
	if len(recs) == 0 {
		resp.Sentence = "I couldn't find any wines like that. "
		if getBudget() < 5000 {
			resp.Sentence += "Should we look among the more expensive bottles?"
			resp.State["state"] = StateRecommendationsAlterBudget
		} else {
			resp.Sentence += "Should we expand your search to more wines?"
			resp.State["state"] = StateRecommendationsAlterQuery
		}
		return nil
	}
	log.Println("showing product")
	offset := getOffset()
	product := recs[offset]
	var size string
	product.Size = strings.TrimSpace(strings.ToLower(product.Size))
	if len(product.Size) > 0 && product.Size != "750ml" {
		size = fmt.Sprintf(" (%s)", product.Size)
	}
	tmp := fmt.Sprintf("A %s%s for $%.2f. ", product.Name, size,
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
		tmp += "Does that sound good"
	case 2:
		tmp += "Should I add it to your cart"
	}
	if len(getSelectedProducts()) > 0 {
		r = rand.Intn(6)
		switch r {
		case 0:
			tmp += " as well?"
		case 1:
			tmp += " too?"
		case 2:
			tmp += " also?"
		case 3, 4, 5:
			tmp += "?"
		}
	} else {
		tmp += "?"
	}
	if product.Stock > 1 {
		val := product.Stock
		if val > 12 {
			val = 12
		}
		r = rand.Intn(2)
		switch r {
		case 0:
			tmp += fmt.Sprintf(" You can order up to %d of them.",
				val)
		case 1:
			tmp += fmt.Sprintf(" You can get 1 to %d of them.", val)
		}
	}
	resp.Sentence = language.SuggestedProduct(tmp, offset)
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

func getSelectedProducts() dt.ProductSels {
	products, ok := resp.State["productsSelected"].([]dt.ProductSel)
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

func removeSelectedProduct(name string) {
	log.Println("removing", name, "from cart")
	prods := getSelectedProducts()
	var success bool
	for i, prod := range prods {
		if name == prod.Name {
			resp.State["productsSelected"] = append(prods[:i],
				prods[i+1:]...)
			log.Println("removed", name)
			success = true
		}
	}
	if !success {
		log.Println("failed to remove", name, "from", prods)
	}
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
	state, ok := resp.State["state"].(float64)
	if !ok {
		state = 0.0
	}
	return state
}
