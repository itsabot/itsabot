package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/avabot/pkg"
	"github.com/garyburd/go-oauth/oauth"
)

type Yelp int

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
		Rating       int
		Location     struct {
			Address string
			City    string
		}
	}
}

var credPath = flag.String(
	"config",
	"config.json",
	"Path to configuration file containing the application's credentials.")

func main() {
	p := pkg.NewPackage("yelp", ":4001")
	if err := p.Register(); err != nil {
		log.Fatalln(err)
	}
	// TODO: Handle contractions (e.g. "where's") and plurals in Ava itself
	si := avabot.StructuredInput{
		Command: {"find", "where", "show"},
		Object: {
			"food",
			"restaurant",
			"restaurants",
			"pizza",
			"chinese",
			"japanese",
			"korean",
			"asian",
			"italian",
			"ramen",
			"eat",
			// Perhaps extend and move to a separate txt file
		},
	}
	if err := p.RespondTo(si); err != nil {
		log.Fatalln(err)
	}
	var c client
	if err := readCredentials(&c); err != nil {
		log.Fatalln(err)
	}
}

// TODO: Add support for custom sorting, locations
func (t *Yelp) search(query string, offset int) {
	form := url.Values{
		"term":     query,
		"location": {"Santa Monica, CA"},
		"limit":    1,
	}
	var data response
	err := c.get("http://api.yelp.com/v2/search", form, &data)
	if err != nil {
		log.Fatal(err)
	}
	for _, b := range data.Businesses {
		addr := ""
		if len(b.Location.DisplayAddress) > 0 {
			addr = b.Location.DisplayAddress[0]
		}
		log.Println(b.Name, addr)
	}
}

func readCredentials(c *client) error {
	b, err := ioutil.ReadFile(*credPath)
	if err != nil {
		return err
	}
	var creds struct {
		ConsumerKey    string
		ConsumerSecret string
		Token          string
		TokenSecret    string
	}
	if err := json.Unmarshal(b, &creds); err != nil {
		return err
	}
	c.client.Credentials.Token = creds.ConsumerKey
	c.client.Credentials.Secret = creds.ConsumerSecret
	c.token.Token = creds.Token
	c.token.Secret = creds.TokenSecret
	return nil
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
