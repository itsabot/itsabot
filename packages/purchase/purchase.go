// Package purchase enables purchase of goods and services within Ava.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/shared/auth"
	"github.com/avabot/ava/shared/database"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/search"
	"github.com/avabot/ava/shared/sms"
)

type Purchase string

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var db *sqlx.DB
var ec *search.ElasticClient
var tc *twilio.Client

// resp enables the Run() function to skip to the FollowUp function if basic
// requirements are met.
var resp *datatypes.Response

const (
	StateNone float64 = iota
	StatePreferences
	StateBudget
	StateRecommendations
	StateProductSelection
	StateShippingAddress
	StateAuthorizing
	StatePurchase
	StateComplete
)

func main() {
	flag.Parse()
	var err error
	db, err = database.ConnectDB()
	if err != nil {
		log.Fatalln(err)
	}
	ec = search.NewClient()
	tc = sms.NewClient()
	trigger := &datatypes.StructuredInput{
		Commands: language.Purchase(),
		Objects:  language.Alcohol(),
	}
	p, err := pkg.NewPackage("purchase", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	purchase := new(Purchase)
	if err := p.Register(purchase); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (t *Purchase) Run(m *datatypes.Message,
	respMsg *datatypes.ResponseMsg) error {
	resp = m.NewResponse()
	resp.State = map[string]interface{}{
		"state":            StateNone,             // maintains state
		"query":            "",                    // search query
		"budget":           "",                    // suggested price
		"recommendations":  []datatypes.Product{}, // search results
		"offset":           uint(0),               // index in search
		"shippingAddress":  &datatypes.Address{},
		"productsSelected": []datatypes.Product{},
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

func (t *Purchase) FollowUp(m *datatypes.Message,
	respMsg *datatypes.ResponseMsg) error {
	if resp == nil {
		if err := m.GetLastResponse(db); err != nil {
			return err
		}
		resp = m.LastResponse
	}
	// have we already made the purchase?
	if resp.State["state"].(float64) == StateComplete {
		// if so, reset state to allow for other purchases
		return t.Run(m, respMsg)
	}
	// TODO allow the user to direct the conversation, e.g. say "something
	// more expensive" and have Ava respond appropriately

	// if purchase has not been made, move user through the package's states
	err := updateState(m, resp, respMsg)
	if err != nil {
		return err
	}
	return pkg.SaveResponse(respMsg, resp)
}

func updateState(m *datatypes.Message, resp *datatypes.Response,
	respMsg *datatypes.ResponseMsg) error {
	switch resp.State["state"].(float64) {
	case StatePreferences:
		// TODO ensure Ava remembers past answers for preferences
		q := resp.State["query"].(string)
		q += " " + m.Input.Sentence
		resp.State["query"] = q
		resp.State["state"] = StateBudget
		resp.Sentence = "Ok. How much do you usually pay for a " +
			"bottle of wine?"
	case StateBudget:
		// TODO ensure Ava remembers past answers for budget
		log.Println("SENTENCE", m.Input.Sentence)
		val, budget, err := language.ExtractCurrency(m.Input.Sentence)
		if err != nil {
			return err
		}
		if budget == nil {
			return pkg.SaveResponse(respMsg, resp)
		}
		resp.State["budget"] = val
		resp.State["state"] = StateRecommendations
		// skip to StateRecommendations
		return updateState(m, resp, respMsg)
	case StateRecommendations:
		query := resp.State["query"].(string)
		if len(query) == 0 {
			log.Println("err:",
				"seeking recommendations without query")
		}
		log.Println("QUERY", query)
		results, err := ec.FindProducts(query, "alcohol", 20)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			resp.Sentence = "I'm sorry. I couldn't find anything " +
				"like that."
			return pkg.SaveResponse(respMsg, resp)
		}
		// TODO - better recommendations
		// results = sales.SortByRecommendation(results)
		resp.State["recommendations"] = results
		resp.State["state"] = StateProductSelection
		log.Println("OFFSET =", resp.State["offset"])
		if err = recommendProduct(resp); err != nil {
			return err
		}
	case StateProductSelection:
		// was the recommendation Ava made good?
		yes := language.ExtractYesNo(m.Input.Sentence)
		if yes == nil {
			return handleKeywords(m, resp, respMsg)
		}
		if !*yes {
			resp.State["offset"] = resp.State["offset"].(uint) + 1
			return pkg.SaveResponse(respMsg, resp)
		}
		selection, err := currentSelection(resp.State)
		if err != nil {
			return err
		}
		resp.State["productsSelected"] = append(
			resp.State["productsSelected"].([]datatypes.Product),
			*selection)
		resp.State["state"] = StateShippingAddress
		// skip to StateShippingAddress
		return updateState(m, resp, respMsg)
	case StateShippingAddress:
		// TODO add memory of shipping addresses
		addr, err := language.ExtractAddress(m.Input.Sentence)
		if err != nil {
			return err
		}
		if addr == nil {
			return pkg.SaveResponse(respMsg, resp)
		}
		if err := m.User.SaveAddress(db, addr); err != nil {
			return err
		}
		resp.State["shippingAddress"] = addr
		resp.State["state"] = StateAuthorizing
		price := float64(resp.State["price"].(uint64)) / 100
		tmp := fmt.Sprintf("%s. Should I place the order?", price)
		resp.Sentence = fmt.Sprintf("Ok. It comes to %s", tmp)
	case StateAuthorizing:
		yes := language.ExtractYesNo(m.Input.Sentence)
		if yes == nil {
			return pkg.SaveResponse(respMsg, resp)
		}
		if !*yes {
			return handleKeywords(m, resp, respMsg)
		}
		// TODO complete
	case StatePurchase:
		// TODO ensure Ava follows up to ensure the
		err := auth.Purchase(db, tc, auth.MethodZip, m,
			resp.State["productsSelected"].([]datatypes.Product))
		if err != nil {
			return err
		}
		resp.State["state"] = StateComplete
		resp.Sentence = "Great! I've placed the order. You'll receive a confirmation by email."
	}
	return pkg.SaveResponse(respMsg, resp)
}

func currentSelection(state map[string]interface{}) (*datatypes.Product,
	error) {
	recs := state["recommendations"].([]datatypes.Product)
	l := uint(len(recs))
	if l == 0 {
		return &datatypes.Product{}, errors.New("empty recommendations")
	}
	offset := state["offset"].(uint)
	if l <= offset {
		err := errors.New("offset exceeds recommendation length")
		return &datatypes.Product{}, err
	}
	return &recs[offset], nil
}

func handleKeywords(m *datatypes.Message, resp *datatypes.Response,
	respMsg *datatypes.ResponseMsg) error {
	words := strings.Fields(m.Input.Sentence)
	for _, word := range words {
		switch word {
		case "detail", "details", "description", "more about", "review",
			"rating", "rated":
		case "price", "cost", "shipping", "how much":
		case "similar", "else", "different":
			resp.State["offset"] = resp.State["offset"].(uint) + 1
			if err := recommendProduct(resp); err != nil {
				return err
			}
		}
	}
	return pkg.SaveResponse(respMsg, resp)
}

func recommendProduct(resp *datatypes.Response) error {
	results := resp.State["results"].([]datatypes.Product)
	product := results[resp.State["offset"].(uint)]
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
	return nil
}
