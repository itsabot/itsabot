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
	trigger := &dt.StructuredInput{
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

func (t *Yelp) Run(m *dt.Msg, respMsg *dt.RespMsg) error {
	// NOTE optional: get state before this package was run, enabling
	// chaining. For instance, allow passing context from a FourSquare
	// result into this. See the opening lines of FollowUp() for an example
	resp := m.NewResponse()
	resp.State = map[string]interface{}{
		"query":      "",
		"location":   "",
		"offset":     float64(0),
		"businesses": []interface{}{},
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
			return pkg.SaveResponse(respMsg, resp)
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
			return pkg.SaveResponse(respMsg, resp)
		}
		resp.State["location"] = loc.Name
	}
	if err := t.searchYelp(resp); err != nil {
		log.Println(err)
	}
	return pkg.SaveResponse(respMsg, resp)
}

// FollowUp handles dialog question/answers and additional user queries
func (t *Yelp) FollowUp(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	// Retrieve the conversation's context
	if err := m.GetLastResponse(db); err != nil {
		log.Println("err getting last response")
		return err
	}
	resp := m.NewResponse()

	// First we handle dialog. If we asked for a location, use the response
	log.Printf("state %+v\n", resp.State)
	if resp.State["location"] == "" {
		loc := m.Input.StructuredInput.All()
		resp.State["location"] = loc
		if err := t.searchYelp(resp); err != nil {
			log.Println(err)
		}
		return pkg.SaveResponse(respMsg, resp)
	}

	// If no businesses are returned inform the user now
	log.Println("businesses", resp.State["businesses"])
	if resp.State["businesses"] != nil &&
		len(resp.State["businesses"].([]interface{})) == 0 {
		resp.Sentence = "I couldn't find anything like that"
		return pkg.SaveResponse(respMsg, resp)
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
			s = fmt.Sprintf("You can reach them here: %s",
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
		case "menu", "have":
			s = fmt.Sprintf("Yelp might have a menu... %s",
				getURL(resp, offI))
			resp.Sentence = s
		case "not", "else", "no", "anything", "something":
			resp.State["offset"] = float64(offI + 1)
			if err := t.searchYelp(resp); err != nil {
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
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	return pkg.SaveResponse(respMsg, resp)
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

func (t *Yelp) searchYelp(resp *dt.Resp) error {
	q := resp.State["query"].(string)
	loc := resp.State["location"].(string)
	offset := resp.State["offset"].(float64)
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
