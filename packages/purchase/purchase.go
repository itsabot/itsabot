// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
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
var vocab dt.Vocab
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
	p.Vocab = dt.NewVocab(
		dt.VocabHandler{
			Fn:       kwDetail,
			WordType: "Object",
			Words: []string{"detail", "description", "review",
				"rating", "about"},
		},
		dt.VocabHandler{
			Fn:       kwPrice,
			WordType: "Object",
			Words:    []string{"price", "cost", "shipping", "total"},
		},
		dt.VocabHandler{
			Fn:       kwSearch,
			WordType: "Command",
			Words:    []string{"find", "search", "show", "give"},
		},
		dt.VocabHandler{
			Fn:       kwNextProduct,
			WordType: "Object",
			Words: []string{"similar", "else", "different",
				"looking", "another", "recommend", "next"},
		},
		dt.VocabHandler{
			Fn:       kwMoreExpensive,
			WordType: "Object",
			Words: []string{"more expensive", "event", "nice",
				"nicer"},
		},
		dt.VocabHandler{
			Fn:       kwLessExpensive,
			WordType: "Object",
			Words: []string{"less expensive", "cheaper", "cheap",
				"pricey"},
		},
		dt.VocabHandler{
			Fn:       kwCart,
			WordType: "Object",
			Words:    []string{"cart"},
		},
		dt.VocabHandler{
			Fn:       kwCheckout,
			WordType: "Command",
			Words: []string{"checkout", "check out", "done",
				"ready", "ship"},
		},
		dt.VocabHandler{
			Fn:       kwRemoveFromCart,
			WordType: "Command",
			Words:    []string{"remove", "rid", "drop"},
		},
		dt.VocabHandler{
			Fn:       kwHelp,
			WordType: "Command",
			Words:    []string{"help", "command"},
		},
		dt.VocabHandler{
			Fn:       kwStop,
			WordType: "Command",
			Words:    []string{"stop"},
		},
	)
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
		setState(StateRedWhite)
		return p.SaveResponse(respMsg, resp)
	}
	resp.State["query"] = query
	resp.State["category"] = cat
	if len(query) <= 8 {
		setState(StateCheckPastPreferences)
	} else {
		if err := prefs.Save(ctx.DB, resp.UserID, pkgName,
			prefs.KeyTaste, query); err != nil {
			l.Errorln("saving taste pref", err)
		}
		currency := language.ExtractCurrency(m.Input.Sentence)
		if currency.Valid && currency.Int64 > 0 {
			l.WithField("value", currency.Int64).Debugln(
				"currency valid and > 0")
			resp.State["budget"] = uint64(currency.Int64)
			setState(StateSetRecommendations)
			err := prefs.Save(ctx.DB, resp.UserID, pkgName,
				prefs.KeyBudget,
				strconv.FormatInt(currency.Int64, 10))
			if err != nil {
				l.Errorln("saving budget pref", err)
			}
		} else {
			setState(StatePreferences)
		}
	}
	if err := updateState(m, resp, respMsg); err != nil {
		l.Errorln(err)
	}
	return p.SaveResponse(respMsg, resp)
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
	var kw bool
	var err error
	if getState() > StateSetRecommendations {
		kw, err = handleKeywords(ctx, resp, m.Stems)
		if err != nil {
			return err
		}
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
			return nil
		}
		l.WithField("cat", getCategory()).Infoln("selected category")
		setState(StateCheckPastPreferences)
		return updateState(m, resp, respMsg)
	case StateCheckPastPreferences:
		tastePref, err := prefs.Get(ctx.DB, resp.UserID, pkgName,
			prefs.KeyTaste)
		if err != nil {
			return err
		}
		if len(tastePref) == 0 {
			setState(StatePreferences)
			resp.Sentence = "Ok. What do you usually look for in a wine? (e.g. dry, fruity, sweet, earthy, oak, etc.)"
			return nil
		}
		resp.State["query"] = tastePref
		setState(StateCheckPastBudget)
		return updateState(m, resp, respMsg)
	case StatePreferences:
		resp.State["query"] = getQuery() + " " + m.Input.Sentence
		if getBudget() == 0 {
			setState(StateCheckPastBudget)
			if err := prefs.Save(ctx.DB, resp.UserID, pkgName,
				prefs.KeyTaste, getQuery()); err != nil {
				l.Errorln("saving taste pref", err)
				return err
			}
			return updateState(m, resp, respMsg)
		}
		setState(StateSetRecommendations)
		return updateState(m, resp, respMsg)
	case StateCheckPastBudget:
		budgetPref, err := prefs.Get(ctx.DB, resp.UserID, pkgName,
			prefs.KeyBudget)
		if err != nil {
			return err
		}
		l.WithField("budgetPref", budgetPref).Debugln("got budgetPref")
		if len(budgetPref) > 0 {
			resp.State["budget"], err = strconv.ParseUint(
				budgetPref, 10, 64)
			if err != nil {
				return err
			}
			l.WithField("budget", getBudget()).Debugln("got budget")
			if getBudget() > 0 {
				setState(StateSetRecommendations)
				return updateState(m, resp, respMsg)
			}
		}
		l.Debugln("updating state to StateBudget")
		setState(StateBudget)
		resp.Sentence = "Ok. How much do you usually pay for a bottle?"
	case StateBudget:
		val := language.ExtractCurrency(m.Input.Sentence)
		if !val.Valid {
			return nil
		}
		resp.State["budget"] = uint64(val.Int64)
		setState(StateSetRecommendations)
		err := prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyBudget,
			strconv.FormatUint(getBudget(), 10))
		if err != nil {
			l.Errorln("saving budget pref", err)
			return err
		}
		fallthrough
	case StateSetRecommendations:
		err := setRecs(resp)
		if err != nil {
			l.Errorln("setting recs", err)
			return err
		}
		setState(StateMakeRecommendation)
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
		setState(StateSetRecommendations)
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
		setState(StateSetRecommendations)
		err := prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyBudget,
			strconv.FormatUint(getBudget(), 10))
		if err != nil {
			return err
		}
		return updateState(m, resp, respMsg)
	case StateMakeRecommendation:
		if err := recommendProduct(resp); err != nil {
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
			setState(StateMakeRecommendation)
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
				setState(StateRecommendationsAlterBudget)
			} else {
				resp.Sentence += "Should we expand your search to more wines?"
				setState(StateRecommendationsAlterQuery)
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
		setState(StateContinueShopping)
	case StateContinueShopping:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if !yes.Valid {
			return nil
		}
		if yes.Bool {
			resp.State["offset"] = getOffset() + 1
			setState(StateMakeRecommendation)
		} else {
			setState(StateShippingAddress)
		}
		return updateState(m, resp, respMsg)
	case StateShippingAddress:
		prods := getSelectedProducts()
		if len(prods) == 0 {
			resp.Sentence = "You haven't picked any products. Should we keep looking?"
			setState(StateContinueShopping)
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
		setState(StatePurchase)
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
		setState(StateAuth)
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
		setState(StateComplete)
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

func handleKeywords(ctx *dt.Ctx, resp *dt.Resp, stems []string) (bool, error) {
	// TODO move thanks, thank you to Ava core
	err := p.Vocab.HandleKeywords(ctx, resp, stems)
	if err == dt.ErrNoFn {
		if getState() != StateProductSelection {
			currency := language.ExtractCurrency(resp.Sentence)
			if currency.Valid && currency.Int64 > 0 {
				setBudget(uint64(currency.Int64))
				setState(StateSetRecommendations)
			}
		}
	}
	return len(resp.Sentence) > 0, err
}

func recommendProduct(resp *dt.Resp) error {
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
			setState(StateRecommendationsAlterBudget)
		} else {
			resp.Sentence += "Should we expand your search to more wines?"
			setState(StateRecommendationsAlterQuery)
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
	setState(StateProductSelection)
	return nil
}

func setRecs(resp *dt.Resp) error {
	results, err := ctx.EC.FindProducts(getQuery(), getCategory(),
		"alcohol", getBudget())
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

func kwDetail(_ *dt.Ctx, _ int) (string, error) {
	var s string
	r := rand.Intn(3)
	switch r {
	case 0:
		s = "Every wine I recommend is at the top of its craft."
	case 1:
		s = "I only recommend the best."
	case 2:
		s = "This wine has been personally selected by leading wine experts."
	}
	return s, nil
}

func kwPrice(_ *dt.Ctx, _ int) (string, error) {
	var s string
	prods := getSelectedProducts()
	if len(prods) == 0 {
		s = "Shipping is around $12 for the first bottle + $1.20 for every bottle after."
		return s, nil
	}
	prices := prods.Prices(getShippingAddress())
	s = fmt.Sprintf("The items cost $%.2f, ",
		float64(prices["products"])/100)
	s += fmt.Sprintf("shipping is $%.2f, ", float64(prices["shipping"])/100)
	if prices["tax"] > 0.0 {
		s += fmt.Sprintf("and tax is $%.2f, ",
			float64(prices["tax"])/100)
	}
	s += fmt.Sprintf("totaling $%.2f.", float64(prices["total"])/100)
	return s, nil
}

func kwSearch(ctx *dt.Ctx, _ int) (string, error) {
	setOffset(0)
	setQuery(ctx.Msg.Input.StructuredInput.Objects.String())
	err := prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyTaste,
		ctx.Msg.Input.StructuredInput.Objects.String())
	if err != nil {
		return "", err
	}
	cat := extractWineCategory(ctx.Msg.Input.Sentence)
	if len(cat) == 0 {
		setState(StateRedWhite)
	} else {
		setCategory(cat)
		setState(StateSetRecommendations)
	}
	return "", nil
}

func kwNextProduct(_ *dt.Ctx, _ int) (string, error) {
	setOffset(float64(getOffset() + 1))
	setState(StateMakeRecommendation)
	return "", nil
}

func kwLessExpensive(ctx *dt.Ctx, mod int) (string, error) {
	mod *= -1
	return kwMoreExpensive(ctx, mod)
}

func kwMoreExpensive(ctx *dt.Ctx, mod int) (string, error) {
	budg := getBudget()
	var tmp int
	if budg >= 10000 {
		tmp = int(budg) + (7500 * mod)
	} else if budg >= 5000 {
		tmp = int(budg) + (3500 * mod)
	} else {
		tmp = int(budg) + (2000 * mod)
	}
	if tmp <= 0 {
		tmp = 1000
	}
	setBudget(uint64(tmp))
	setState(StateSetRecommendations)
	err := prefs.Save(ctx.DB, resp.UserID, pkgName, prefs.KeyBudget,
		strconv.Itoa(tmp))
	return "", err
}

func kwCart(_ *dt.Ctx, _ int) (string, error) {
	var s string
	prods := getSelectedProducts()
	var prodNames []string
	for _, prod := range prods {
		var count string
		if prod.Count > 1 {
			count = fmt.Sprintf("%dx", prod.Count)
		}
		name := fmt.Sprintf("%s (%s$%.2f)", prod.Name, count,
			float64(prod.Price)/100)
		prodNames = append(prodNames, name)
	}
	if len(prods) == 0 {
		s = "You haven't picked any wines, yet."
	} else if len(prods) == 1 {
		s = "You've picked a " + prodNames[0] + "."
	} else {
		s = fmt.Sprintf(
			"You've picked %d wines: ", len(prods))
		s += language.SliceToString(prodNames, "and") + "."
	}
	if len(prods) > 0 {
		tmp := " Let me know when you're ready to checkout."
		// 255 is the database varchar limit, but we should aim
		// to be below 140 (sms)
		if len(s) > 140-len(tmp) {
			// 4 refers to the length of the ellipsis
			s = s[0 : 140-len(tmp)-4]
			s += "... "
		}
		s += tmp
	}
	return s, nil
}

func kwCheckout(_ *dt.Ctx, _ int) (string, error) {
	if tskAddr != nil {
		tskAddr.ResetState()
	}
	if tskPurch != nil {
		tskPurch.ResetState()
	}
	setState(StateShippingAddress)
	return "", nil
}

func kwRemoveFromCart(ctx *dt.Ctx, _ int) (string, error) {
	var s string
	prods := getSelectedProducts()
	var prodNames []string
	for _, prod := range prods {
		prodNames = append(prodNames, prod.Name)
	}
	var matches []string
	for _, w := range strings.Fields(ctx.Msg.Input.Sentence) {
		if len(w) <= 3 {
			continue
		}
		tmp := fuzzy.FindFold(w, prodNames)
		if len(tmp) > 0 {
			matches = append(matches, tmp...)
		}
	}
	if len(matches) == 0 {
		s = "I couldn't find a wine like that in your cart."
	} else if len(matches) == 1 {
		s = fmt.Sprintf("Ok, I'll remove the %s.", matches[0])
		removeSelectedProduct(matches[0])
	} else {
		s = "Ok, I'll remove those."
		for _, match := range matches {
			removeSelectedProduct(match)
		}
	}
	r := rand.Intn(2)
	switch r {
	case 0:
		s += " Is there something else I can help you find?"
	case 1:
		s += " Would you like to find another?"
	}
	setState(StateContinueShopping)
	return s, nil
}

func kwHelp(_ *dt.Ctx, _ int) (string, error) {
	s := "At any time you can ask to see your cart, checkout, find something different (dry, fruity, earthy, etc.), or find something more or less expensive."
	return s, nil
}

func kwStop(_ *dt.Ctx, _ int) (string, error) {
	return "Ok.", nil
}
