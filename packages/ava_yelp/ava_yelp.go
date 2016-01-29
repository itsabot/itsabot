package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/garyburd/go-oauth/oauth"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/knowledge"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/nlp"
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

var ErrNoBusinesses = errors.New("no businesses")

var c client
var db *sqlx.DB
var p *pkg.Pkg
var l *log.Entry

func main() {
	var coreaddr string
	flag.StringVar(&coreaddr, "coreaddr", "",
		"Port used to communicate with Ava.")
	flag.Parse()

	c.client.Credentials.Token = os.Getenv("YELP_CONSUMER_KEY")
	c.client.Credentials.Secret = os.Getenv("YELP_CONSUMER_SECRET")
	c.token.Token = os.Getenv("YELP_TOKEN")
	c.token.Secret = os.Getenv("YELP_TOKEN_SECRET")

	var err error
	db, err = pkg.ConnectDB()
	if err != nil {
		l.Fatalln(err)
	}

	trigger := &nlp.StructuredInput{
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
	p, err = pkg.NewPackage("ava_yelp", coreaddr, trigger)
	if err != nil {
		l.Fatalln("building", err)
	}
	yelp := new(Yelp)
	if err := p.Register(yelp); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Yelp) Run(m *dt.Msg, respMsg *dt.RespMsg) error {
	m.State = map[string]interface{}{
		"query":      "",
		"location":   "",
		"offset":     float64(0),
		"businesses": []interface{}{},
	}
	si := m.StructuredInput
	query := ""
	for _, o := range si.Objects {
		query += o + " "
	}
	for _, p := range si.Places {
		query += p + " "
	}
	m.State["query"] = query
	if len(si.Places) == 0 {
		l.Infoln("no place entered, getting location")
		loc, question, err := knowledge.GetLocation(db, m.User)
		if err != nil {
			return err
		}
		if len(question) > 0 {
			if loc != nil && len(loc.Name) > 0 {
				m.State["location"] = loc.Name
			}
			m.Sentence = question
			return p.SaveMsg(respMsg, m)
		}
		m.State["location"] = loc.Name
	}
	// Occurs in the case of "nearby" or other contextual place terms, where
	// no previous context was available to expand it.
	if len(m.State["location"].(string)) == 0 {
		loc, question, err := knowledge.GetLocation(db, m.User)
		if err != nil {
			return err
		}
		if len(question) > 0 {
			if loc != nil && len(loc.Name) > 0 {
				m.State["location"] = loc.Name
			}
			m.Sentence = question
			return p.SaveMsg(respMsg, m)
		}
		m.State["location"] = loc.Name
	}
	if err := t.searchYelp(m); err != nil {
		l.WithField("fn", "searchYelp").Errorln(err)
	}
	return p.SaveMsg(respMsg, m)
}

// FollowUp handles dialog question/answers and additional user queries
func (t *Yelp) FollowUp(m *dt.Msg, respMsg *dt.RespMsg) error {
	// First we handle dialog. If we asked for a location, use the response
	if m.State["location"] == "" {
		loc := m.StructuredInput.All()
		m.State["location"] = loc
		if err := t.searchYelp(m); err != nil {
			l.WithField("fn", "searchYelp").Errorln(err)
		}
		return p.SaveMsg(respMsg, m)
	}

	// If no businesses are returned inform the user now
	if m.State["businesses"] != nil &&
		len(m.State["businesses"].([]interface{})) == 0 {
		m.Sentence = "I couldn't find anything like that"
		return p.SaveMsg(respMsg, m)
	}

	// Responses were returned, and the user has asked this package an
	// additional query. Handle the query by keyword
	words := strings.Fields(m.Sentence)
	offI := int(m.State["offset"].(float64))
	var s string
	for _, w := range words {
		w = strings.TrimRight(w, ").,;?!:")
		switch strings.ToLower(w) {
		case "rated", "rating", "review", "recommend", "recommended":
			s = fmt.Sprintf("It has a %s star review on Yelp",
				getRating(m, offI))
			m.Sentence = s
		case "number", "phone":
			s = getPhone(m, offI)
			m.Sentence = s
		case "call":
			s = fmt.Sprintf("You can reach them here: %s",
				getPhone(m, offI))
			m.Sentence = s
		case "information", "info":
			s = fmt.Sprintf("Here's some more info: %s",
				getURL(m, offI))
			m.Sentence = s
		case "where", "location", "address", "direction", "directions",
			"addr":
			s = fmt.Sprintf("It's at %s", getAddress(m, offI))
			m.Sentence = s
		case "pictures", "pic", "pics":
			s = fmt.Sprintf("I found some pics here: %s",
				getURL(m, offI))
			m.Sentence = s
		case "menu", "have":
			s = fmt.Sprintf("Yelp might have a menu... %s",
				getURL(m, offI))
			m.Sentence = s
		case "not", "else", "no", "anything", "something":
			m.State["offset"] = float64(offI + 1)
			if err := t.searchYelp(m); err != nil {
				l.WithField("fn", "searchYelp").Errorln(err)
			}
		// TODO perhaps handle this case and "thanks" at the AVA level?
		// with bayesian classification
		case "good", "great", "yes", "perfect":
			// TODO feed into learning engine
			m.Sentence = language.Positive()
		case "thanks", "thank":
			m.Sentence = language.Welcome()
		}
		if len(m.Sentence) > 0 {
			return p.SaveMsg(respMsg, m)
		}
	}
	return p.SaveMsg(respMsg, m)
}

func getRating(r *dt.Msg, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	return fmt.Sprintf("%.1f", firstBusiness["Rating"].(float64))
}

func getURL(r *dt.Msg, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	return firstBusiness["mobile_url"].(string)
}

func getPhone(r *dt.Msg, offset int) string {
	businesses := r.State["businesses"].([]interface{})
	firstBusiness := businesses[offset].(map[string]interface{})
	return firstBusiness["display_phone"].(string)
}

func getAddress(r *dt.Msg, offset int) string {
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
	resp, err := c.client.Get(nil, &c.token, urlStr, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("yelp status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (t *Yelp) searchYelp(m *dt.Msg) error {
	q := m.State["query"].(string)
	loc := m.State["location"].(string)
	offset := m.State["offset"].(float64)
	l.WithFields(log.Fields{
		"q":      q,
		"loc":    loc,
		"offset": offset,
	}).Infoln("searching yelp")
	form := url.Values{
		"term":     {q},
		"location": {loc},
		"limit":    {fmt.Sprintf("%.0f", offset+1)},
	}
	var data yelpResp
	err := c.get("http://api.yelp.com/v2/search", form, &data)
	if err != nil {
		/*
			m.Sentence = "I can't find that for you now. " +
				"Let's try again later."
			l.WithField("fn", "get").Errorln(err)
			return err
		*/
		// return for confused response, given Yelp errors are rare, but
		// unintentional runs of Yelp queries are much more common
		return nil
	}
	m.State["businesses"] = data.Businesses
	if len(data.Businesses) == 0 {
		m.Sentence = "I couldn't find any places like that nearby."
		return nil
	}
	offI := int(offset)
	if len(data.Businesses) <= offI {
		m.Sentence = "That's all I could find."
		return nil
	}
	b := data.Businesses[offI]
	addr := ""
	if len(b.Location.DisplayAddress) > 0 {
		addr = b.Location.DisplayAddress[0]
	}
	if offI == 0 {
		m.Sentence = "Ok. How does this place look? " + b.Name +
			" at " + addr
	} else {
		m.Sentence = fmt.Sprintf("What about %s instead?", b.Name)
	}
	return nil
}
