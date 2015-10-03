package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
	"github.com/garyburd/go-oauth/oauth"
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

var credPath = flag.String(
	"config",
	"ava_modules/yelp/config.json",
	"Path to configuration file containing the application's credentials.")
var port = flag.Int("port", 0, "Port used to communicate with Ava.")

var c client
var plog *log.Entry

func main() {
	if os.Getenv("AVA_ENV") == "production" {
		log.SetLevel(log.WarnLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
	plog = log.WithField("package", "yelp")
	c.client.Credentials.Token = os.Getenv("YELP_CONSUMER_KEY")
	c.client.Credentials.Secret = os.Getenv("YELP_CONSUMER_SECRET")
	c.token.Token = os.Getenv("YELP_TOKEN")
	c.token.Secret = os.Getenv("YELP_TOKEN_SECRET")
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
	p, err := pkg.NewPackage("yelp", trigger)
	if err != nil {
		plog.Fatal("creating package", p.Config.Name, err)
	}
	yelp := new(Yelp)
	if err := p.Register(yelp); err != nil {
		plog.Fatal("registering package ", err)
	}
}

func (t *Yelp) Run(si *datatypes.StructuredInput, resp *string) error {
	plog.Debug("package called")
	var query, location string
	for _, o := range si.Objects {
		query += o + " "
	}
	for _, p := range si.Places {
		query += p + " "
	}
	// TODO: Get location if unknown
	if len(si.Places) == 0 {
		location = "Santa Monica"
	}
	r, err := t.search(query, location, 0)
	if err != nil {
		plog.Error("search yelp: ", err)
		r = "I couldn't run that for you at this time."
	}
	*resp = r
	return nil
}

func (t *Yelp) FollowUp(si *datatypes.StructuredInput, resp *string) error {
	plog.Debug("package called as follow up")
	plog.Debug(si.String())
	*resp = "OK!"
	return nil
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
	plog.Println(b.Name, addr)
	return "How does this place look? " + b.Name + " at " + addr, nil
}

func (c *client) get(urlStr string, params url.Values, v interface{}) error {
	resp, err := c.client.Get(nil, &c.token, urlStr, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		plog.Error(resp)
		return fmt.Errorf("yelp status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
