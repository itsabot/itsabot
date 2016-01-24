// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/renstrom/fuzzysearch/fuzzy"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/nlp"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/prefs"
	"github.com/avabot/ava/shared/task"
)

type Purchase string

var ErrEmptyRecommendations = errors.New("empty recommendations")
var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var vocab dt.Vocab
var db *sqlx.DB
var ec *dt.SearchClient
var p *pkg.Pkg
var sm *dt.StateMachine
var l *log.Entry

// m enables the Run() function to skip to the FollowUp function if basic
// requirements are met.
var m *dt.Msg = &dt.Msg{}

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
	log.SetLevel(log.DebugLevel)
	l = log.WithFields(log.Fields{
		"pkg": pkgName,
	})
	rand.Seed(time.Now().UnixNano())
	var err error
	db, err = pkg.ConnectDB()
	if err != nil {
		l.Fatalln(err)
	}
	ec = dt.NewSearchClient()
	trigger := &nlp.StructuredInput{
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
				"rating"},
		},
		dt.VocabHandler{
			Fn:       kwPrice,
			WordType: "Object",
			Words: []string{"price", "cost", "shipping",
				"total"},
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
			Words:    []string{"more expensive", "event", "nice"},
		},
		dt.VocabHandler{
			Fn:       kwLessExpensive,
			WordType: "Object",
			Words:    []string{"less expensive", "cheap", "pricey"},
		},
		dt.VocabHandler{
			Fn:       kwSearch,
			WordType: "Command",
			Words:    []string{"find", "search", "show", "give"},
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
	sm, err = dt.NewStateMachine(pkgName)
	if err != nil {
		l.Errorln(err)
		return
	}
	sm.SetStates(
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					return "Are you looking for a red or white? We can also find sparkling wines or rose."
				},
				OnInput: func(in *dt.Msg) {
					c := extractWineCategory(in.Sentence)
					sm.SetMemory(in, "category", c)
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return sm.HasMemory(in, "category"), ""
				},
			},
			{
				Memory: "taste",
				OnEntry: func(in *dt.Msg) string {
					// Conversational things, like "Ok", "got it",
					// etc. are added automatically questions when
					// appropriate (TODO)
					return "What do you usually look for in a wine? (e.g. dry, fruity, sweet, earthy, oak, etc.)"
				},
				OnInput: func(in *dt.Msg) {
					s := in.StructuredInput.Objects.String() +
						" wine"
					sm.SetMemory(in, "taste", s)
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return sm.HasMemory(in, "taste"), ""
				},
			},
			{
				Memory: "budget",
				OnEntry: func(in *dt.Msg) string {
					return "How much do you usually pay for a bottle?"
				},
				OnInput: func(in *dt.Msg) {
					val := language.ExtractCurrency(in.Sentence)
					if !val.Valid {
						return
					}
					u := uint64(val.Int64)
					sm.SetMemory(in, "budget", u)
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return sm.HasMemory(in, "budget"), ""
				},
			},
			{
				OnEntry: func(in *dt.Msg) string {
					q := sm.GetMemory(in, "taste").String()
					cat := sm.GetMemory(in, "category").String()
					bdg := sm.GetMemory(in, "budget").Int64()
					results, err := ec.FindProducts(q, cat,
						"alcohol", uint64(bdg))
					if err != nil {
						l.Errorln("findproducts", err)
					}
					var s string
					if len(results) == 0 {
						// TODO
						/*
							q, cat, bdg = fixSearchParams(q, cat,
								bdg)
							results = ec.FindProducts(q, cat,
								"alcohol", bdg)
							s = "Here's the closest I could find. "
						*/
					}
					sm.SetMemory(in, "selected_products", results)
					tmp, err := recommendProduct(&results[0])
					if err != nil {
						l.Errorln(err)
						return ""
					}
					return s + tmp
				},
				// Here sProductSelection is moved elsewhere because
				// it's a little long. The OnInput function here keeps
				// track of the viewed product index and increments
				// based on user feedback until something is selected
				OnInput: sProductSelection,
				Complete: func(in *dt.Msg) (bool, string) {
					if sm.HasMemory(in, "selected_products") {
						if sm.HasMemory(in, "selection_finished") {
							return true, ""
						}
					}
					return false, ""
				},
			},
		},
		task.New(sm, task.RequestAddress),
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					prods := getSelectedProducts()
					addr := getShippingAddress()
					p := float64(prods.Prices(addr)["total"]) / 100
					s := fmt.Sprintf("It comes to %2f. ", p)
					return s + "Should I place the order?"
				},
				OnInput: func(in *dt.Msg) {
					yes := language.ExtractYesNo(in.Sentence)
					if yes.Valid && yes.Bool {
						sm.SetMemory(in, "purchase_confirmed", true)
					}
				},
				// TODO
				// If the user responds with "No" above, Ava determines
				// whether to reply with confusion or accept the answer,
				// e.g. "Ok." based on the user's language automatically
				Complete: func(in *dt.Msg) (bool, string) {
					return sm.HasMemory(in, "purchase_confirmed"), ""
				},
			},
		},
		task.New(sm, task.RequestPurchaseAuthZip),
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					return "Got it. Should be on its way soon!"
				},
			},
		},
	)
	sm.SetDBConn(db)
	sm.SetLogger(l)
	sm.SetOnReset(func(in *dt.Msg) {
		sm.SetMemory(in, "query", "")
		sm.SetMemory(in, "category", "")
		sm.SetMemory(in, "budget", "")
		sm.SetMemory(in, "offset", "")
		sm.SetMemory(in, "purchase_confirmed", "")
		sm.SetMemory(in, "recommendations", nil)
		sm.SetMemory(in, "current_shipping_address", nil)
		sm.SetMemory(in, "selected_products", nil)
		sm.SetMemory(in, "purchase", false)
	})
	purchase := new(Purchase)
	if err := p.Register(purchase); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Purchase) Run(m *dt.Msg, respMsg *dt.RespMsg) error {
	respMsg.Sentence = sm.Next(m)
	return p.SaveMsg(respMsg, m)
}

func sProductSelection(in *dt.Msg) {
	// was the recommendation Ava made good?
	yes := language.ExtractYesNo(in.Sentence)
	if !yes.Valid {
		return
	}
	if !yes.Bool {
		// TODO convert to new API
		// setOffset(getOffset() + 1)
		return
	}
	count := language.ExtractCount(in.Sentence)
	if count.Valid {
		if count.Int64 == 0 {
			// asked to order 0 wines. trigger confused
			// reply
			return
		}
	}
	mem := sm.GetMemory(in, "recommendations")
	var prods []*dt.Product
	if err := json.Unmarshal(mem.Val, &prods); err != nil {
		l.Errorln(err)
	}
	/*
		if err == ErrEmptyRecommendations {
			m.Sentence = "I couldn't find any wines like that. "
			if getBudget() < 5000 {
				m.Sentence += "Should we look among the more expensive bottles?"
				setState(StateRecommendationsAlterBudget)
			} else {
				m.Sentence += "Should we expand your search to more wines?"
				setState(StateRecommendationsAlterQuery)
			}
			neturn updateState(in, out)
		}
		if err != nil {
			l.Errorln("getting current selection", err)
		}
	*/
	// TODO
	/*
		if !count.Valid || count.Int64 <= 1 {
			count.Int64 = 1
			m.Sentence = "Ok, I've added it to your cart. Should we look for a few more?"
		} else if uint(count.Int64) > selection.Stock {
			m.Sentence = "I'm sorry, but I don't have that many available. Should we do "
			return
		} else {
			m.Sentence = fmt.Sprintf(
				"Ok, I'll add %d to your cart. Should we look for a few more?",
				count.Int64)
		}
		prod := dt.ProductSel{
			Product: selection,
			Count:   uint(count.Int64),
		}
		var prods []dt.Product
		mem := sm.GetMemory(in, "selected_products")
		if err = json.Unmarshal(mem.Val, prods); err != nil {
			l.Errorln("retrieving selected_products", err)
			return
		}
		sm.SetMemory(in, "selected_products", append(prods, prod))
	*/
}

func recommendProduct(p *dt.Product) (string, error) {
	recs := getRecommendations()
	offset := getOffset()
	if len(recs) == 0 || int(offset) >= len(recs) {
		var s string
		if len(recs) == 0 {
			s = "I couldn't find any wines like that. "
		} else {
			s = "I'm out of wines in that category. "
		}
		if getBudget() < 5000 {
			s += "Should we look among the more expensive bottles?"
		} else {
			s += "Should we expand your search to more wines?"
		}
		return s, nil
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
		return "", err
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
	return language.SuggestedProduct(tmp, offset), nil
}

func setRecs(m *dt.Msg) error {
	results, err := ec.FindProducts(getQuery(), getCategory(), "alcohol",
		getBudget())
	if err != nil {
		return err
	}
	if len(results) == 0 {
		m.Sentence = "I'm sorry. I couldn't find anything like that."
	}
	// TODO - better recommendations
	// results = sales.SortByRecommendation(results)
	m.State["recommendations"] = results
	return nil
}

func removeSelectedProduct(name string) {
	l.WithField("product", name).Infoln("removing from cart")
	prods := getSelectedProducts()
	var success bool
	for i, prod := range prods {
		if name == prod.Name {
			m.State["productsSelected"] = append(prods[:i],
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

func kwDetail(_ *dt.Msg, _ int) error {
	r := rand.Intn(3)
	switch r {
	case 0:
		m.Sentence = "Every wine I recommend is at the top of its craft."
	case 1:
		m.Sentence = "I only recommend the best."
	case 2:
		m.Sentence = "This wine has been personally selected by leading wine experts."
	}
	return nil
}

func kwPrice(_ *dt.Msg, _ int) error {
	prods := getSelectedProducts()
	if len(prods) == 0 {
		m.Sentence = "Shipping is around $12 for the first bottle + $1.20 for every bottle after."
		return nil
	}
	prices := prods.Prices(getShippingAddress())
	s := fmt.Sprintf("The items cost $%.2f, ",
		float64(prices["products"])/100)
	s += fmt.Sprintf("shipping is $%.2f, ", float64(prices["shipping"])/100)
	if prices["tax"] > 0.0 {
		s += fmt.Sprintf("and tax is $%.2f, ",
			float64(prices["tax"])/100)
	}
	s += fmt.Sprintf("totaling $%.2f.", float64(prices["total"])/100)
	m.Sentence = s
	return nil
}

func kwSearch(in *dt.Msg, _ int) error {
	l.Debugln("hit kwSearch")
	setOffset(0)
	setQuery(in.StructuredInput.Objects.String())
	err := prefs.Save(db, in.User.ID, pkgName, prefs.KeyTaste,
		in.StructuredInput.Objects.String())
	if err != nil {
		return err
	}
	cat := extractWineCategory(in.Sentence)
	if len(cat) == 0 {
		setState(StateRedWhite)
	} else {
		setCategory(cat)
		setState(StateSetRecommendations)
	}
	return nil
}

func kwNextProduct(_ *dt.Msg, _ int) error {
	setOffset(float64(getOffset() + 1))
	setState(StateMakeRecommendation)
	return nil
}

func kwLessExpensive(in *dt.Msg, mod int) error {
	mod *= -1
	return kwMoreExpensive(in, mod)
}

func kwMoreExpensive(in *dt.Msg, mod int) error {
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
	err := prefs.Save(db, in.User.ID, pkgName, prefs.KeyBudget,
		strconv.Itoa(tmp))
	return err
}

func kwCart(_ *dt.Msg, _ int) error {
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
	m.Sentence = s
	return nil
}

func kwCheckout(_ *dt.Msg, _ int) error {
	l.Debugln("here...")
	setState(StateShippingAddress)
	return nil
}

func kwRemoveFromCart(in *dt.Msg, _ int) error {
	var s string
	prods := getSelectedProducts()
	var prodNames []string
	for _, prod := range prods {
		prodNames = append(prodNames, prod.Name)
	}
	var matches []string
	for _, w := range strings.Fields(m.Sentence) {
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
	m.Sentence = s
	return nil
}

func kwHelp(_ *dt.Msg, _ int) error {
	m.Sentence = "At any time you can ask to see your cart, checkout, find something different (dry, fruity, earthy, etc.), or find something more or less expensive."
	return nil
}

func kwStop(_ *dt.Msg, _ int) error {
	l.Debugln("hit kwStop")
	m.Sentence = "Ok."
	return nil
}
