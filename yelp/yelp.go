package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"net/url"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
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
			Address        string
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
	plog = log.WithField("package", "yelp")
	if err := readCredentials(&c); err != nil {
		plog.Fatalln(err)
	}
	// TODO: Handle contractions (e.g. "where's") and plurals in Ava itself
	trigger := &datatypes.StructuredInput{
		Command: []string{
			"find",
			"where",
			"show",
			"recommend",
			"recommendation",
			"recommendations",
		},
		Objects: language.Foods(),
	}
	p, err := pkg.NewPackage("yelp", "", 4001, trigger)
	if err != nil {
		plog.Fatal("creating package", p.Config.Name, err)
	}
	if err := p.Register(); err != nil &&
		err.Error() != "gob: type rpc.Client has no exported fields" {
		plog.Fatal("registering package", p.Config.Name, err)
	}
	bootRPCServer(4002)
	plog.Debug("booted " + p.Config.Name)
}

func (t *Yelp) Run(si *datatypes.StructuredInput, resp *string) error {
	var query, location string
	for _, o := range si.Objects {
		query += o + " "
	}
	for _, p := range si.Places {
		query += p + " "
	}
	r, err := t.search(query, location, 0)
	if err != nil {
		// TODO: Save log
		r = "I couldn't run that for you at this time."
	}
	plog.Debug("yelp api response: " + r)
	resp = &r
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

func readCredentials(c *client) error {
	plog.Debug("credential path")
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

func bootRPCServer(port int) {
	yelp := new(Yelp)
	if err := rpc.Register(yelp); err != nil {
		plog.Fatalln(err)
	}
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		plog.Fatalln("rpc listen:", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			plog.Fatalln("rpc accept:", err)
		}
		go rpc.ServeConn(conn)
	}
}
