// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/renstrom/fuzzysearch/fuzzy"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/nlp"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/task"
)

type Purchase string

var ErrEmptyRecommendations = errors.New("empty recommendations")
var vocab dt.Vocab
var db *sqlx.DB
var ec *dt.SearchClient
var p *pkg.Pkg
var sm *dt.StateMachine
var l *log.Entry

// m enables the Run() function to skip to the FollowUp function if basic
// requirements are met.
var m *dt.Msg = &dt.Msg{}

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
	var coreaddr string
	flag.StringVar(&coreaddr, "coreaddr", "",
		"Port used to communicate with Ava.")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	l = log.WithFields(log.Fields{"pkg": pkgName})
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
	p, err = pkg.NewPackage(pkgName, coreaddr, trigger)
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
	sm = dt.NewStateMachine(pkgName)
	sm.SetStates(
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					l.Debugln("onentry")
					return "Are you looking for a red or white? We can also find sparkling wines or rose."
				},
				OnInput: func(in *dt.Msg) {
					l.Debugln("oninput")
					c := extractWineCategory(in.Sentence)
					sm.SetMemory(in, "category", c)
				},
				Complete: func(in *dt.Msg) (bool, string) {
					l.Debugln("complete")
					return sm.HasMemory(in, "category"), ""
				},
			},
			{
				SkipIfComplete: true,
				OnEntry: func(in *dt.Msg) string {
					// Conversational things, like "Ok", "got it",
					// etc. are added automatically questions when
					// appropriate (TODO)
					return "What do you usually look for in a wine? (e.g. dry, fruity, sweet, earthy, oak, etc.)"
				},
				OnInput: func(in *dt.Msg) {
					s := in.StructuredInput.Objects.
						String() + " wine"
					sm.SetMemory(in, "taste", s)
					l.Debugln("set taste to", s)
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return sm.HasMemory(in, "taste"), ""
				},
			},
			{
				SkipIfComplete: true,
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
					l.Debugln("set budget to", u)
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return sm.HasMemory(in, "budget"), ""
				},
			},
			{
				//Label: "recommendations",
				OnEntry: func(in *dt.Msg) string {
					log.Println("recs on entry")
					q := sm.GetMemory(in, "taste").String()
					log.Println("taste", q)
					cat := sm.GetMemory(in, "category").String()
					log.Println("category", cat)
					bdg := sm.GetMemory(in, "budget").Int64()
					log.Println("budget", bdg)
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
					sm.SetMemory(in, "recommendations", results)
					if len(results) > 0 {
						tmp, err := recommendProduct(in, &results[0])
						l.Debugln("here end")
						if err != nil {
							l.Errorln(err)
							return ""
						}
						return s + tmp
					} else {
						return "I couldn't find anything like that."
					}
				},
				OnInput: func(in *dt.Msg) {
					// was the recommendation Ava made good?
					log.Println("recs on input")
					yes := language.ExtractYesNo(in.Sentence)
					if !yes.Valid {
						return
					}
					if !yes.Bool {
						sm.SetMemory(in, "offset", sm.GetMemory(in, "offset").Int64()+1)
						return
					}
					count := language.ExtractCount(in.Sentence)
					if count.Valid {
						if count.Int64 == 0 {
							// asked to order 0 wines.
							return
						}
					}
					var prods []dt.Product
					mem := sm.GetMemory(in, "selected_products")
					err := json.Unmarshal(mem.Val, prods)
					if err != nil {
						l.Errorln("unmarshaling selected products", err)
					}
					recs := getRecommendations(in)
					offset := int(sm.GetMemory(in, "offset").Int64())
					if len(recs) <= offset {
						l.Errorln("recs shorter than offset")
						return
					}
					prods = append(prods, recs[offset])
					sm.SetMemory(in, "selected_products", prods)
					sm.SetMemory(in, "recently_added", true)
				},
				// NOTE everything below this point is a WIP
				Complete: func(in *dt.Msg) (bool, string) {
					log.Println("recs complete")
					if sm.HasMemory(in, "selected_products") {
						if sm.HasMemory(in, "selection_finished") {
							return true, ""
						}
					}
					return false, ""
				},
			},
			{
				OnEntry: func(in *dt.Msg) string {
					added := sm.GetMemory(in, "recently_added").Bool()
					if added {
						return "Ok. I've added it to your cart. Should we keep looking?"
					}
					sm.SetMemory(in, "recently_added", false)
					// The user didn't want to add the item
					// to his cart. SetState will then check
					// if this state is Complete() before
					// continuing
					return sm.SetState(in, "shipping_address")
				},
				OnInput: func(in *dt.Msg) {
					yes := language.ExtractYesNo(in.Sentence)
					if !yes.Valid {
						return
					}
					if !yes.Bool {
						sm.SetMemory(in, "offset", sm.GetMemory(in, "offset").Int64()+1)
						return
					}
				},
				Complete: func(in *dt.Msg) (bool, string) {
					prods := getSelectedProducts(in)
					return len(prods) > 0, "not implemented"
				},
			},
		},
		task.New(sm, task.RequestAddress, "shipping_address"),
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					prods := getSelectedProducts(in)
					addr := getShippingAddress(in)
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
		task.New(sm, task.RequestPurchaseAuthZip, "request_purchase"),
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
		l.Debugln("resetting")
		sm.SetMemory(in, "query", "")
		sm.SetMemory(in, "category", "")
		sm.SetMemory(in, "budget", "")
		sm.SetMemory(in, "offset", 0)
		sm.SetMemory(in, "purchase_confirmed", "")
		sm.SetMemory(in, "recommendations", []byte{})
		sm.SetMemory(in, "current_shipping_address", []byte{})
		sm.SetMemory(in, "selected_products", []byte{})
		sm.SetMemory(in, "purchase", false)
	})
	purchase := new(Purchase)
	if err := p.Register(purchase); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Purchase) Run(in *dt.Msg, resp *string) error {
	sm.Reset(in)
	return t.FollowUp(in, resp)
}

func (t *Purchase) FollowUp(in *dt.Msg, resp *string) error {
	*resp = p.Vocab.HandleKeywords(in)
	if len(*resp) == 0 {
		*resp = sm.Next(in)
	}
	return nil
}

func recommendProduct(in *dt.Msg, p *dt.Product) (string, error) {
	recs := getRecommendations(in)
	offset := int(sm.GetMemory(in, "offset").Int64())
	if len(recs) == 0 || offset >= len(recs) {
		var s string
		if len(recs) == 0 {
			s = "I couldn't find any wines like that. "
		} else {
			s = "I'm out of wines in that category. "
		}
		if getBudget(in) < 5000 {
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
	if len(getSelectedProducts(in)) > 0 {
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
	return language.SuggestedProduct(tmp, uint(offset)), nil
}

func removeSelectedProduct(in *dt.Msg, name string) {
	l.WithField("product", name).Infoln("removing from cart")
	prods := getSelectedProducts(in)
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

func kwDetail(_ *dt.Msg, _ int) string {
	r := rand.Intn(3)
	var s string
	switch r {
	case 0:
		s = "Every wine I recommend is at the top of its craft."
	case 1:
		s = "I only recommend the best."
	case 2:
		s = "This wine has been personally selected by leading wine experts."
	}
	return s
}

func kwPrice(in *dt.Msg, _ int) string {
	prods := getSelectedProducts(in)
	if len(prods) == 0 {
		return "Shipping is around $12 for the first bottle + $1.20 for every bottle after."
	}
	prices := prods.Prices(getShippingAddress(in))
	s := fmt.Sprintf("The items cost $%.2f, ",
		float64(prices["products"])/100)
	s += fmt.Sprintf("shipping is $%.2f, ", float64(prices["shipping"])/100)
	if prices["tax"] > 0.0 {
		s += fmt.Sprintf("and tax is $%.2f, ",
			float64(prices["tax"])/100)
	}
	s += fmt.Sprintf("totaling $%.2f.", float64(prices["total"])/100)
	return s
}

func kwSearch(in *dt.Msg, _ int) string {
	l.Debugln("hit kwSearch")
	sm.SetMemory(in, "offset", 0)
	sm.SetMemory(in, "query", in.StructuredInput.Objects.String())
	cat := extractWineCategory(in.Sentence)
	if len(cat) == 0 {
		sm.Reset(in)
		return ""
	}
	sm.SetMemory(in, "category", cat)
	return sm.SetState(in, "recommendations")
}

func kwNextProduct(in *dt.Msg, _ int) string {
	offset := sm.GetMemory(in, "offset")
	sm.SetMemory(in, "offset", offset.Int64()+1)
	return sm.SetState(in, "recommendations")
}

func kwLessExpensive(in *dt.Msg, mod int) string {
	mod *= -1
	return kwMoreExpensive(in, mod)
}

func kwMoreExpensive(in *dt.Msg, mod int) string {
	budg := uint64(sm.GetMemory(in, "budget").Int64())
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
	sm.SetMemory(in, "budget", uint64(tmp))
	return sm.SetState(in, "recommendations")
}

func kwCart(in *dt.Msg, _ int) string {
	prods := getSelectedProducts(in)
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
	var s string
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
	return s
}

func kwCheckout(in *dt.Msg, _ int) string {
	return sm.SetState(in, "shipping_address")
}

func kwRemoveFromCart(in *dt.Msg, _ int) string {
	prods := getSelectedProducts(in)
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
	var s string
	if len(matches) == 0 {
		s = "I couldn't find a wine like that in your cart."
	} else if len(matches) == 1 {
		s = fmt.Sprintf("Ok, I'll remove the %s.", matches[0])
		removeSelectedProduct(in, matches[0])
	} else {
		s = "Ok, I'll remove those."
		for _, match := range matches {
			removeSelectedProduct(in, match)
		}
	}
	return s
}

func kwHelp(_ *dt.Msg, _ int) string {
	return "At any time you can ask to see your cart, checkout, find something different (dry, fruity, earthy, etc.), or find something more or less expensive."
}

func kwStop(_ *dt.Msg, _ int) string {
	l.Debugln("hit kwStop")
	return "Ok."
}

func getSelectedProducts(in *dt.Msg) dt.ProductSels {
	prods := dt.ProductSels{}
	mem := sm.GetMemory(in, "selected_products")
	err := json.Unmarshal(mem.Val, &prods)
	if err != nil {
		l.Errorln("getSelectedProducts", err)
	}
	return prods
}

func getRecommendations(in *dt.Msg) []dt.Product {
	prods := []dt.Product{}
	mem := sm.GetMemory(in, "recommendations")
	if err := json.Unmarshal(mem.Val, &prods); err != nil {
		l.Errorln("getRecommendations", err)
	}
	return prods
}

func getShippingAddress(in *dt.Msg) *dt.Address {
	addr := dt.Address{}
	mem := sm.GetMemory(in, "shipping_address")
	if err := json.Unmarshal(mem.Val, &addr); err != nil {
		l.Errorln("getRecommendations", err)
	}
	return &addr
}

func getBudget(in *dt.Msg) uint64 {
	mem := sm.GetMemory(in, "budget")
	return uint64(mem.Int64())
}
