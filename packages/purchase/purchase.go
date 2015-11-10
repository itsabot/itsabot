package main

import (
	"flag"
	"log"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/database"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
)

type Purchase string

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var db *sqlx.DB

func main() {
	flag.Parse()
	var err error
	db, err = database.connectDB()
	if err != nil {
		log.Fatalln(err)
	}
	trigger := &datatypes.StructuredInput{
		Commands: []string{
			"find",
			"buy",
			"purchase",
			"recommend",
			"recommendation",
			"recommendations",
		},
		Objects: language.Wines(),
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
	resp := m.NewResponse()
	resp.State = map[string]interface{}{
		"query":           "",
		"budget":          "",
		"recommendations": "",
		"offset":          uint(0),
		"shippingAddress": "",
		"time":            "",
		"price":           float64(0),
	}
	si := m.Input.StructuredInput
	query := ""
	for _, o := range si.Objects {
		query += o + " "
	}
	for _, p := range si.Places {
		resp.State["shippingAddress"] += p + " "
	}

	// We want to ignore simple queries like wine, and have the user tell
	// us more about what they like.
	if resp.State["query"].(string).length <= 5 {
		resp.Sentence = "What kind of wine are you looking for?"
	}

	results, err := search.Find(resp.State["query"], 20)
	if err != nil {
		return err
	}
	resp.State["recommendations"] = results

	// Establish what the user's looking for. Consider pairing and past
	// favorites.

	// Establish how much they're willing to pay.
	if resp.State["budget"].(string).length == 0 {
		sales.WillingnessToPay()
	}

	// Feed query into a recommendation engine

	// Where will it be shipped? Offer to ship if within an acceptable
	// state. If not, then offer to find a wine shop near them that may be
	// able to help them locate it, passing this request to the Yelp
	// package.

	// If shipping, is there a specific time they need it delivered?

	// Upsell engine. Frequently purchased with, discounts for bulk, etc.
	sales.Upsell()

	return nil
}
