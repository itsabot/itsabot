package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/garyburd/go-oauth/oauth"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/knowledge"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
)

type Yelp string

type client struct {
	client oauth.Client
	token  oauth.Credentials
}

type response struct {
	Businesses []struct {
		Name         string
		ImageUrl     string `json:"image_url"`
		MobileUrl    string `json:"mobile_url"`
		DisplayPhone string `json:"display_phone"`
		Distance     int
		Rating       float64
		Location     struct {
			City           string
			DisplayAddress []string `json:"display_address"`
		}
	}
}

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var ErrNoBusinesses = errors.New("no businesses")

var c client
var db *sqlx.DB

func main() {
	flag.Parse()
	c.client.Credentials.Token = os.Getenv("YELP_CONSUMER_KEY")
	c.client.Credentials.Secret = os.Getenv("YELP_CONSUMER_SECRET")
	c.token.Token = os.Getenv("YELP_TOKEN")
	c.token.Secret = os.Getenv("YELP_TOKEN_SECRET")
	db = connectDB()
	trigger := &datatypes.StructuredInput{
		Commands: []string{
			"find",
			"where",
			"show",
			"recommend",
			"recommendation",
			"recommendations",
		},
		Objects: language.Foods(),
	}
	p, err := pkg.NewPackage("yelp", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	yelp := new(Yelp)
	if err := p.Register(yelp); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (t *Yelp) Run(m *datatypes.Message, resp *string) error {
	log.Println("package called")
	var query, location string
	si := m.Input.StructuredInput
	for _, o := range si.Objects {
		query += o + " "
	}
	for _, p := range si.Places {
		query += p + " "
	}
	if len(si.Places) == 0 {
		loc, err := knowledge.LastLocation(db, m.User)
		if err != nil {
			log.Println("err: getting last location")
			return err
		}
		location = loc.Name
	}
	r, err := t.search(query, location, 0)
	if err != nil {
		log.Println("err: search yelp: ", err)
		r = "I couldn't run that for you at this time."
	}
	*resp = r
	return nil
}

// TODO: Build a way to set up an expected state or response
func (t *Yelp) FollowUp(m *datatypes.Message, resp *string) error {
	for _, o := range m.Input.StructuredInput.Objects {
		switch o {
		case "rating", "review", "recommend":
			rating, err := getRating(m)
			if err != nil {
				return err
			}
			*resp = "It has a " + rating + " review on Yelp."
			return nil
		case "number", "phone", "call":
		case "information":
		}
	}
	return nil
}

func getRating(m *datatypes.Message) (string, error) {
	r := datatypes.Response{}
	if err := m.LastResponse(db, &r); err != nil {
		return "", err
	}
	log.Println("STATE: ", r.State)
	return "5", nil
	/*
		if len(r.State.Businesses) == 0 {
			return "", ErrNoBusinesses
		}
		return fmt.Sprintf("%.1f", r.State.Businesses[0].Rating), nil
	*/
}

// TODO: Add support for custom sorting, locations
func (t *Yelp) search(query, location string, offset int) (string, error) {
	form := url.Values{
		"term":     {query},
		"location": {location},
		"limit":    {"1"},
	}
	var data response
	err := c.get("http://api.yelp.com/v2/search", form, &data)
	if err != nil {
		return "", err
	}
	if len(data.Businesses) == 0 {
		return "I couldn't find any places like that nearby.", nil
	}
	b := data.Businesses[0]
	addr := ""
	if len(b.Location.DisplayAddress) > 0 {
		addr = b.Location.DisplayAddress[0]
	}
	log.Println(b.Name, addr)
	return "How does this place look? " + b.Name + " at " + addr, nil
}

func (c *client) get(urlStr string, params url.Values, v interface{}) error {
	resp, err := c.client.Get(nil, &c.token, urlStr, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("err:", resp)
		return fmt.Errorf("yelp status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func connectDB() *sqlx.DB {
	log.Println("connecting to db")
	var db *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=egtann dbname=ava sslmode=disable")
	}
	if err != nil {
		log.Println("err: could not connect to db", err)
	}
	return db
}
