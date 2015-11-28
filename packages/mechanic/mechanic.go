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

type Mechanic string

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
var p *pkg.Pkg
var db *sqlx.DB

func main() {
	flag.Parse()
	c.client.Credentials.Token = os.Getenv("YELP_CONSUMER_KEY")
	c.client.Credentials.Secret = os.Getenv("YELP_CONSUMER_SECRET")
	c.token.Token = os.Getenv("YELP_TOKEN")
	c.token.Secret = os.Getenv("YELP_TOKEN_SECRET")
	db = connectDB()
	trigger := &dt.StructuredInput{
		Commands: language.Join(
			language.Recommend(),
			language.Broken(),
			language.Repair(),
		),
		Objects: language.Join(
			language.Vehicles(),
			language.AutomotiveBrands(),
		),
	}
	var err error
	p, err = pkg.NewPackage("mechanic", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	mechanic := new(Mechanic)
	if err := p.Register(mechanic); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (pt *Mechanic) Run(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	resp := m.NewResponse()
	resp.State = map[string]interface{}{
		"query":      "",
		"location":   "",
		"offset":     float64(0),
		"businesses": []interface{}{},
		"warranty":   "",
		"preference": "",
		"brand":      "",
	}
	si := m.Input.StructuredInput
	query := ""
	for _, o := range si.Objects {
		for _, b := range language.AutomotiveBrands() {
			if strings.ToLower(o) == b {
				resp.State["brand"] = b
				break
			}
		}
		query += o + " "
	}
	resp.State["query"] = query
	if len(si.Places) == 0 {
		log.Println("no place entered, getting location")
		loc, question, err := knowledge.GetLocation(db, m.User)
		if err != nil {
			return err
		}
		if len(question) > 0 {
			if loc != nil && len(loc.Name) > 0 {
				resp.State["location"] = loc.Name
			}
			resp.Sentence = question
			return p.SaveResponse(respMsg, resp)
		}
		resp.State["location"] = loc.Name
	}
	// Occurs in the case of "nearby" or other contextual place terms, where
	// no previous context was available to expand it.
	if len(resp.State["location"].(string)) == 0 {
		loc, question, err := knowledge.GetLocation(db, m.User)
		if err != nil {
			return err
		}
		if len(question) > 0 {
			if loc != nil && len(loc.Name) > 0 {
				resp.State["location"] = loc.Name
			}
			resp.Sentence = question
			return p.SaveResponse(respMsg, resp)
		}
		resp.State["location"] = loc.Name
	}
	if err := pt.searchYelp(resp); err != nil {
		log.Println(err)
	}
	return p.SaveResponse(respMsg, resp)
}

func (pt *Mechanic) FollowUp(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	// Retrieve the conversation's context
	if err := m.GetLastResponse(db); err != nil {
		log.Println("err getting last response")
		return err
	}
	resp := m.NewResponse()

	// First we handle dialog, filling out the user's location
	if resp.State["location"] == "" {
		loc := m.Input.StructuredInput.All()
		if len(loc) > 0 {
			resp.State["location"] = loc
			resp.Sentence = "Ok. I can help you. " +
				"What kind of car do you drive?"
		}
		return p.SaveResponse(respMsg, resp)
	}

	// Check the automotive brand
	if resp.State["brand"] == "" {
		var brand string
		tmp := m.Input.StructuredInput.Objects
	Loop:
		for _, w1 := range language.AutomotiveBrands() {
			for _, w2 := range tmp {
				if w1 == strings.ToLower(w2) {
					brand = w2
					break Loop
				}
			}
		}
		if len(brand) > 0 {
			resp.State["brand"] = brand
			resp.Sentence = "Is your car still in warranty?"
		}
		return p.SaveResponse(respMsg, resp)
	}

	// Check warranty information
	if resp.State["warranty"] == "" {
		warr := m.Input.StructuredInput.All()
		if language.Yes(warr) {
			resp.State["warranty"] = "yes"
			resp.State["preference"] = "dealer"
			if err := pt.searchYelp(resp); err != nil {
				log.Println(err)
			}
		} else if language.No(warr) {
			resp.State["warranty"] = "no"
			resp.Sentence = "Do you prefer the dealership or a recommended mechanic?"
		}
		return p.SaveResponse(respMsg, resp)
	}

	// Does the user prefer dealerships or mechanics?
	if resp.State["preference"] == "" {
		words := strings.Fields(m.Input.Sentence)
		for _, w := range words {
			if w == "dealer" || w == "dealers" {
				resp.State["preference"] = "dealer"
				break
			} else if w == "mechanic" || w == "mechanics" {
				resp.State["preference"] = "mechanic"
				break
			}
		}
		if resp.State["preference"] != "" {
			if err := pt.searchYelp(resp); err != nil {
				log.Println(err)
			}
		}
		return p.SaveResponse(respMsg, resp)
	}

	// If no businesses are returned inform the user now
	log.Println("businesses", resp.State["businesses"])
	if resp.State["businesses"] != nil &&
		len(resp.State["businesses"].([]interface{})) == 0 {
		resp.Sentence = "I couldn't find anything like that"
		return p.SaveResponse(respMsg, resp)
	}

	// Responses were returned, and the user has asked this package an
	// additional query. Handle the query by keyword
	words := strings.Fields(m.Input.Sentence)
	offI := int(resp.State["offset"].(float64))
	var s string
	for _, w := range words {
		w = strings.TrimRight(w, ").,;?!:")
		switch strings.ToLower(w) {
		case "rated", "rating", "review", "recommend", "recommended":
			s = fmt.Sprintf("It has a %s star review on Yelp",
				getRating(resp, offI))
			resp.Sentence = s
		case "number", "phone":
			s = getPhone(resp, offI)
			resp.Sentence = s
		case "call":
			s = fmt.Sprintf("Try this one: %s",
				getPhone(resp, offI))
			resp.Sentence = s
		case "information", "info":
			s = fmt.Sprintf("Here's some more info: %s",
				getURL(resp, offI))
			resp.Sentence = s
		case "where", "location", "address", "direction", "directions",
			"addr":
			s = fmt.Sprintf("It's at %s", getAddress(resp, offI))
			resp.Sentence = s
		case "pictures", "pic", "pics":
			s = fmt.Sprintf("I found some pics here: %s",
				getURL(resp, offI))
			resp.Sentence = s
		case "not", "else", "no", "anything", "something":
			resp.State["offset"] = float64(offI + 1)
			if err := pt.searchYelp(resp); err != nil {
				log.Println(err)
			}
		// TODO perhaps handle this case and "thanks" at the AVA level?
		// with bayesian classification
		case "good", "great", "yes", "perfect":
			// TODO feed into learning engine
			resp.Sentence = language.Positive()
		case "thanks", "thank":
			resp.Sentence = language.Welcome()
		}
		if len(resp.Sentence) > 0 {
			return p.SaveResponse(respMsg, resp)
		}
	}
	return p.SaveResponse(respMsg, resp)
}

func getRating(r *dt.Resp, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	return fmt.Sprintf("%.1f", firstBusiness["Rating"].(float64))
}

func getURL(r *dt.Resp, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	return firstBusiness["mobile_url"].(string)
}

func getPhone(r *dt.Resp, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	return firstBusiness["display_phone"].(string)
}

func getAddress(r *dt.Resp, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	location := firstBusiness["Location"].(map[string]interface{})
	dispAddr := location["display_address"].([]interface{})
	if len(dispAddr) > 1 {
		str1 := dispAddr[0].(string)
		str2 := dispAddr[1].(string)
		return fmt.Sprintf("%s in %s", str1, str2)
	}
	return dispAddr[0].(string)
}

func (c *client) get(urlStr string, params url.Values, v interface{}) error {
	log.Println(urlStr, params)
	resp, err := c.client.Get(nil, &c.token, urlStr, params)
	if err != nil {
		log.Println("1")
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

func (pt *Mechanic) searchYelp(resp *dt.Resp) error {
	q := resp.State["query"].(string)
	loc := resp.State["location"].(string)
	pref := resp.State["preference"].(string)
	brand := resp.State["brand"].(string)
	offset := resp.State["offset"].(float64)
	if brand != "" {
		q = fmt.Sprintf("%s %s", brand, pref)
	} else {
		q = fmt.Sprintf("%s mechanic", q)
	}
	log.Println("searching yelp", q, loc, offset)
	form := url.Values{
		"term":     {q},
		"location": {loc},
		"limit":    {fmt.Sprintf("%.0f", offset+1)},
	}
	var data yelpResp
	err := c.get("http://api.yelp.com/v2/search", form, &data)
	if err != nil {
		resp.Sentence = "I can't find that for you now. " +
			"Let's try again later."
		return err
	}
	resp.State["businesses"] = data.Businesses
	if len(data.Businesses) == 0 {
		resp.Sentence = "I couldn't find any places like that nearby."
		return nil
	}
	offI := int(offset)
	if len(data.Businesses) <= offI {
		resp.Sentence = "That's all I could find."
		return nil
	}
	b := data.Businesses[offI]
	addr := ""
	if len(b.Location.DisplayAddress) > 0 {
		addr = b.Location.DisplayAddress[0]
	}
	if offI == 0 {
		resp.Sentence = "Ok. How does this place look? " + b.Name +
			" at " + addr
	} else {
		resp.Sentence = fmt.Sprintf("What about %s instead?", b.Name)
	}
	return nil
}
