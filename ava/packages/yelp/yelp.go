package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

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

type yelpResp struct {
	Businesses []struct {
		Name         string
		ImageURL     string `json:"image_url"`
		MobileURL    string `json:"mobile_url"`
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

func (t *Yelp) Run(m *datatypes.Message, resp *datatypes.Response) error {
	resp.State = map[string]interface{}{
		"query":    "",
		"location": "",
	}
	si := m.Input.StructuredInput
	query := ""
	for _, o := range si.Objects {
		query += o + " "
	}
	for _, p := range si.Places {
		query += p + " "
	}
	resp.State["query"] = query
	if len(si.Places) == 0 {
		loc, question, err := knowledge.GetLocation(db, m.User)
		if err != nil {
			return err
		}
		if len(question) > 0 {
			if loc != nil && len(loc.Name) > 0 {
				resp.State["location"] = loc.Name
			}
			resp.Sentence = question
			return nil
		}
		resp.State["location"] = loc.Name
	}
	t.searchYelp(resp)
	return nil
}

// FollowUp handles dialog question/answers and additional user queries
func (t *Yelp) FollowUp(m *datatypes.Message, resp *datatypes.Response) error {
	if err := m.GetLastResponse(db); err != nil {
		return err
	}
	resp = m.LastResponse

	// First we handle dialog. If we asked for a location, use the response
	log.Println("state", resp.State)
	if resp.State["location"] == nil && m.LastResponse.QuestionLanguage() {
		if len(m.Input.StructuredInput.Places) == 0 {
			resp.State["location"] = m.Input.Sentence
		} else {
			loc := strings.Join(m.Input.StructuredInput.Places, " ")
			resp.State["location"] = loc
		}
		t.searchYelp(resp)
		return nil
	}

	// If no businesses are returned inform the user now
	if len(resp.State["Businesses"].([]interface{})) == 0 {
		resp.Sentence = "I couldn't find anything like that"
		return nil
	}

	// Responses were returned, and the user has asked this package an
	// additional query. Handle the query by keyword
	words := strings.Fields(m.Input.Sentence)
	var s string
	for _, w := range words {
		w = strings.TrimRight(w, ").,;?!:")
		switch strings.ToLower(w) {
		case "rating", "review", "recommend", "recommended":
			s = fmt.Sprintf("It has a %s review on Yelp",
				getRating(resp))
		case "number", "phone", "call":
			s = getPhone(resp)
		case "information", "info":
			s = fmt.Sprintf("Here's some more info: %s",
				getURL(resp))
		case "where", "location", "address", "direction", "directions":
			s = fmt.Sprintf("It's at %s", getAddress(resp))
		case "pictures", "pic", "pics":
			s = fmt.Sprintf("I found some pics here: %s",
				getURL(resp))
		case "menu", "have":
			s = fmt.Sprintf("Yelp might have a menu... %s",
				getURL(resp))
		}
		resp.Sentence = s
		if len(resp.Sentence) > 0 {
			return nil
		}
	}
	return nil
}

func getRating(r *datatypes.Response) string {
	businesses := r.State["Businesses"].([]interface{})
	firstBusiness := businesses[0].(map[string]interface{})
	return fmt.Sprintf("%.1f", firstBusiness["Rating"].(float64))
}

func getURL(r *datatypes.Response) string {
	businesses := r.State["Businesses"].([]interface{})
	firstBusiness := businesses[0].(map[string]interface{})
	return firstBusiness["MobileUrl"].(string)
}

func getPhone(r *datatypes.Response) string {
	businesses := r.State["Businesses"].([]interface{})
	firstBusiness := businesses[0].(map[string]interface{})
	return firstBusiness["DisplayPhone"].(string)
}

func getAddress(r *datatypes.Response) string {
	businesses := r.State["Businesses"].([]interface{})
	firstBusiness := businesses[0].(map[string]interface{})
	location := firstBusiness["Location"].(map[string]interface{})
	return location["DisplayAddress"].(string)
}

// TODO: Add support for custom sorting, locations
func (t *Yelp) search(query, location string, offset int) (string, error) {
	form := url.Values{
		"term":     {query},
		"location": {location},
		"limit":    {"1"},
	}
	var data yelpResp
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

func (t *Yelp) searchYelp(resp *datatypes.Response) {
	r, err := t.search(resp.State["query"].(string),
		resp.State["location"].(string), 0)
	if err != nil {
		log.Println("err: search yelp", err)
		r = "I can't find that for you now. Let's try again later."
	}
	resp.Sentence = r
}
