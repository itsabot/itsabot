// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
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
var l *log.Entry
var tskAddr *task.Task
var tskPurch *task.Task

// resp enables the Run() function to skip to the FollowUp function if basic
// requirements are met.
var resp *dt.Resp

const (
	StateNone float64 = iota
	StateRedWhite
	StateCheckPastPreferences
	StatePreferences
	StateCheckPastBudget
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
	"AL": true,
	"AK": true,
	"AZ": true,
	"CA": true,
	"CO": true,
	"DC": true,
	"HI": true,
	"ID": true,
	"LA": true,
	"MO": true,
	"NE": true,
	"NV": true,
	"NH": true,
	"NM": true,
	"NY": true,
	"ND": true,
	"OR": true,
	"WI": true,
	"WY": true,
}

// TODO add support for upselling and promotions. Then a Task interface for
// follow ups

func main() {
	flag.Parse()
	l = log.WithFields(log.Fields{
		"pkg": pkgName,
	})
	rand.Seed(time.Now().UnixNano())
	var err error
	ctx, err = dt.NewContext()
	if err != nil {
		l.Fatalln(err)
	}
	trigger := &dt.StructuredInput{
		Commands: language.Purchase(),
		Objects:  language.Alcohol(),
	}
	p, err = pkg.NewPackage(pkgName, *port, trigger)
	if err != nil {
		l.Fatalln("building", err)
	}
	purchase := new(Purchase)
	if err := p.Register(purchase); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Purchase) Run(m *dt.Msg, respMsg *dt.RespMsg) error {
	ctx.Msg = m
	resp = m.NewResponse()
	resp.State = map[string]interface{}{
		"state":            StateNone,        // maintains state
		"query":            "",               // search query
		"category":         "",               // red, white, etc.
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
	cat := extractWineCategory(m.Input.Sentence)
	if len(cat) == 0 {
		resp.Sentence = "Sure. Are you looking for a red or white?"
		resp.State["state"] = StateRedWhite
		return p.SaveResponse(respMsg, resp)
	}
	resp.State["query"] = query
	resp.State["category"] = cat
	resp.State["state"] = StateCheckPastPreferences
	return updateState(m, resp, respMsg)
}

func (t *Purchase) FollowUp(m *dt.Msg, respMsg *dt.RespMsg) error {
	ctx.Msg = m
	if resp == nil {
		if err := m.GetLastResponse(ctx.DB); err != nil {
			return err
		}
		resp = m.LastResponse
	}
	resp.Sentence = ""
	l.Debugln("starting state", getState())
	// have we already made the purchase?
	if getState() == StateComplete {
		// if so, reset state to allow for other purchases
		return t.Run(m, respMsg)
	}
	// allow the user to direct the conversation, e.g. say "something more
	// expensive" and have Ava respond appropriately
	kw, err := handleKeywords(m, resp, respMsg)
	if err != nil {
		return err
	}
	l.WithField("found", kw).Debugln("keywords handled")
	if !kw {
		// if purchase has not been made, move user through the
		// package's states
		if err := updateState(m, resp, respMsg); err != nil {
			return err
		}
	}
	return p.SaveResponse(respMsg, resp)
}

func updateState(m *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) error {
	state := getState()
	switch state {
	case StateRedWhite:
		resp.State["category"] = extractWineCategory(m.Input.Sentence)
		if getCategory() == "" {
			resp.Sentence = "I'm not sure I understand. Are you looking for red, white, rose, or champagne?"
			return nil
		}
		l.WithField("cat", getCategory()).Infoln("selected category")
		resp.State["state"] = StateCheckPastPreferences
		return updateState(m, resp, respMsg)
	case StateCheckPastPreferences:
		tastePref, err := prefs.Get(ctx.DB, resp.UserID, pkgName,
			prefs.KeyTaste)
		if err != nil {
			return err
		}
		if len(tastePref) == 0 {
			resp.State["state"] = StatePreferences
			resp.Sentence = "Ok. What do you usually look for in a wine? (e.g. dry, fruity, sweet, earthy, oak, etc.)"
			return nil
		}
		resp.State["query"] = tastePref
		resp.State["state"] = StateCheckPastBudget
		return updateState(m, resp, respMsg)
	case StatePreferences:
		resp.State["query"] = getQuery() + " " + m.Input.Sentence
		if getBudget() == 0 {
			resp.State["state"] = StateCheckPastBudget
			if err := prefs.Save(ctx.DB, resp.UserID, pkgName,
				prefs.KeyTaste, getQuery()); err != nil {
				l.Errorln("saving budget pref", err)
				return err
			}
			return updateState(m, resp, respMsg)
		}
		resp.State["state"] = StateSetRecommendations
		return updateState(m, resp, respMsg)
	case StateCheckPastBudget:
		budgetPref, err := prefs.Get(ctx.DB, resp.UserID, pkgName,
			prefs.KeyBudget)
		if err != nil {
			return err
		}
		if len(budgetPref) > 0 {
			resp.State["budget"], err = strconv.ParseUint(
				budgetPref, 10, 64)
			if err != nil {
				return err
			}
			resp.State["state"] = StateSetRecommendations
			return updateState(m, resp, respMsg)
		}
		resp.State["state"] = StateBudget
		resp.Sentence = "Ok. How much do you usually pay for a bottle?"
	case StateBudget:
		val, err := language.ExtractCurrency(m.Input.Sentence)
		if err != nil {
			l.Errorln("extracting currency", err)
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
			l.Errorln("saving budget pref", err)
			return err
		}
		fallthrough
	case StateSetRecommendations:
		err := setRecs(resp, respMsg)
		if err != nil {
			l.Errorln("setting recs", err)
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
		if err := recommendProduct(resp, respMsg); err != nil {
			return err
		}
	case StateProductSelection:
		// was the recommendation Ava made good?
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return nil
		}
		if !yes.Bool {
			resp.State["offset"] = getOffset() + 1
			resp.State["state"] = StateMakeRecommendation
			return updateState(m, resp, respMsg)
		}
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
		var err error
		tskAddr, err = task.New(ctx, resp, respMsg)
		if err != nil {
			return err
		}
		done, err := tskAddr.RequestAddress(&addr, len(prods))
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
			return nil
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
		if tskPurch != nil {
			tskPurch.ResetState()
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
		tskPurch, err = task.New(ctx, resp, respMsg)
		if err != nil {
			return err
		}
		done, err := tskPurch.RequestPurchase(task.MethodZip, purchase)
		if err == task.ErrInvalidAuth {
			resp.Sentence = "I'm sorry but that doesn't match what I have. You could try to add a new card here: https://avabot.co/?/cards/new"
			return nil
		}
		if err != nil {
			l.Errorln("requesting purchase", err)
			return err
		}
		if !done {
			l.Infoln("purchase incomplete")
			return nil
		}
		resp.State["state"] = StateComplete
		resp.Sentence = "Great! I've placed the order. You'll receive a confirmation by email."
	}
	return nil
}

func currentSelection(state map[string]interface{}) (*dt.Product, error) {
	recs := getRecommendations()
	ln := uint(len(recs))
	if ln == 0 {
		l.WithFields(log.Fields{
			"q":        getQuery(),
			"offset":   getOffset(),
			"budget":   getBudget(),
			"selProds": len(getSelectedProducts()),
		}).Warnln("empty recs")
		return nil, ErrEmptyRecommendations
	}
	offset := getOffset()
	if ln <= offset {
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
		word = strings.TrimRight(word, ",.?;:!")
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
			prods := getSelectedProducts()
			if len(prods) == 0 {
				resp.Sentence = "Shipping is around $12 for the first bottle + $1.20 for every bottle after."
			} else {
				prices := prods.Prices(getShippingAddress())
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
			}
		case "find", "search", "show", "give":
			resp.State["offset"] = 0
			resp.State["query"] = m.Input.Sentence
			cat := extractWineCategory(m.Input.Sentence)
			if len(cat) > 0 {
				resp.State["category"] = cat
			}
			resp.State["state"] = StateSetRecommendations
			err := prefs.Save(ctx.DB, ctx.Msg.User.ID, pkgName,
				prefs.KeyTaste, m.Input.Sentence)
			if err != nil {
				return false, err
			}
		case "similar", "else", "different", "looking", "look", "another", "recommend", "next":
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
				var count string
				if prod.Count > 1 {
					count = fmt.Sprintf("%dx", prod.Count)
				}
				name := fmt.Sprintf("%s (%s$%.2f)", prod.Name,
					count, float64(prod.Price)/100)
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
				tmp := " Let me know when you're ready to checkout."
				// 255 is the database varchar limit, but we should aim
				// to be below 140 (sms)
				if len(resp.Sentence) > 140-len(tmp) {
					// 4 refers to the length of the ellipsis
					resp.Sentence = resp.Sentence[0 : 140-len(tmp)-4]
					resp.Sentence += "... "
				}
				resp.Sentence += tmp
			}
		case "checkout", "check", "done", "ready":
			if tskAddr != nil {
				tskAddr.ResetState()
			}
			if tskPurch != nil {
				tskPurch.ResetState()
			}
			resp.State["state"] = StateShippingAddress
		case "thanks", "thank":
			resp.Sentence = language.Welcome()
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
					"Ok, I'll remove the %s.", matches[0])
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
		// NOTE too should work with both expensive and cheap, but
		// doesn't yet
		case "less", "too":
			modifier *= -1
		case "much", "very", "extremely":
			modifier *= 2
		}
	}
	if getState() != StateProductSelection {
		currency, err := language.ExtractCurrency(resp.Sentence)
		l.Errorln("extracting currency", err)
		if currency.Valid && currency.Int64 > 0 {
			resp.State["budget"] = currency.Int64
			resp.State["state"] = StateSetRecommendations
		}
	}
	return len(resp.Sentence) > 0, nil
}

func recommendProduct(resp *dt.Resp, respMsg *dt.RespMsg) error {
	recs := getRecommendations()
	offset := getOffset()
	if len(recs) == 0 || int(offset) >= len(recs) {
		if len(recs) == 0 {
			resp.Sentence = "I couldn't find any wines like that. "
		} else {
			resp.Sentence = "I'm out of wines in that category. "
		}
		if getBudget() < 5000 {
			resp.Sentence += "Should we look among the more expensive bottles?"
			resp.State["state"] = StateRecommendationsAlterBudget
		} else {
			resp.Sentence += "Should we expand your search to more wines?"
			resp.State["state"] = StateRecommendationsAlterQuery
		}
		return nil
	}
	product := recs[offset]
	var size string
	product.Size = strings.TrimSpace(strings.ToLower(product.Size))
	if len(product.Size) > 0 && product.Size != "750ml" {
		size = fmt.Sprintf(" (%s)", product.Size)
	}
	tmp := fmt.Sprintf("A %s%s for $%.2f. ", product.Name, size,
		float64(product.Price)/100)
	summary, err := language.Summarize(&product, "products_alcohol")
	if err != nil {
		return err
	}
	tmp += summary + " "
	r := rand.Intn(2)
	switch r {
	case 0:
		tmp += "Does that sound good"
	case 1:
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
	results, err := ctx.EC.FindProducts(getQuery(), getCategory(),
		"alcohol", getBudget(), 20)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		resp.Sentence = "I'm sorry. I couldn't find anything like that."
	}
	for i := range results {
		j := rand.Intn(i + 1)
		results[i], results[j] = results[j], results[i]
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
		l.WithField("type", reflect.TypeOf(resp.State["offset"])).
			Errorln("couldn't get offset: invalid type")
	}
	return uint(0)
}

func getQuery() string {
	return resp.State["query"].(string)
}

func getCategory() string {
	return resp.State["category"].(string)
}

func getBudget() uint64 {
	switch resp.State["budget"].(type) {
	case int64:
		return uint64(resp.State["budget"].(int64))
	case uint64:
		return resp.State["budget"].(uint64)
	case float64:
		return uint64(resp.State["budget"].(float64))
	case string:
		s, err := strconv.ParseUint(resp.State["budget"].(string), 10,
			64)
		if err != nil {
			l.WithField("budget", s).Errorln(
				"couldn't get budget: convert from string")
			s = uint64(0)
		}
		return s
	default:
		l.WithField("type", reflect.TypeOf(resp.State["budget"])).
			Errorln("couldn't get budget: invalid type")
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
			l.Errorln("productsSelected not found",
				resp.State["productsSelected"])
			return nil
		}
		byt, err := json.Marshal(prodMap)
		if err != nil {
			l.Errorln("marshaling products", err)
		}
		if err = json.Unmarshal(byt, &products); err != nil {
			l.Errorln("unmarshaling products", err)
		}
	}
	return products
}

func removeSelectedProduct(name string) {
	l.WithField("product", name).Infoln("removing from cart")
	prods := getSelectedProducts()
	var success bool
	for i, prod := range prods {
		if name == prod.Name {
			resp.State["productsSelected"] = append(prods[:i],
				prods[i+1:]...)
			l.WithField("product", name).Debugln(
				"removed from cart")
			success = true
		}
	}
	if !success {
		l.WithField("name", name).Errorln("failed to remove")
	}
}

func getRecommendations() []dt.Product {
	products, ok := resp.State["recommendations"].([]dt.Product)
	if !ok {
		prodMap, ok := resp.State["recommendations"].(interface{})
		if !ok {
			l.Errorln("recommendations not found",
				resp.State["recommendations"])
			return nil
		}
		byt, err := json.Marshal(prodMap)
		if err != nil {
			l.Errorln("marshaling products", err)
		}
		if err = json.Unmarshal(byt, &products); err != nil {
			l.Errorln("unmarshaling products", err)
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

func extractWineCategory(s string) string {
	s = strings.ToLower(s)
	red := strings.Contains(s, "red")
	var category string
	if red {
		category = "red"
	}
	white := strings.Contains(s, "white")
	if white {
		category = "white"
	}
	rose := strings.Contains(s, "rose")
	if rose {
		category = "rose"
	}
	champagne := strings.Contains(s, "champagne")
	if !champagne {
		champagne = strings.Contains(s, "sparkling")
	}
	if champagne {
		category = "champagne"
	}
	return category
}
