package twilio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// A client manages communication with Twilio API
type Client struct {
	// HTTP client used to communicate with API
	client *http.Client

	// User agent used when communicating with Twilio API
	UserAgent string

	// The Twilio API base URL
	BaseURL *url.URL

	// Credentials which is used for authentication during API request
	AccountSid string
	AuthToken  string

	// Services used for communicating with different parts of the Twilio API
	Messages *MessageService
}

// NewClient returns a new Twilio API client. This will load default http.Client if httpClient is nil.
func NewClient(accountSid, authToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	baseURL, _ := url.Parse(apiBaseURL)

	c := &Client{
		client:     httpClient,
		UserAgent:  userAgent,
		BaseURL:    baseURL,
		AccountSid: accountSid,
		AuthToken:  authToken,
	}

	c.Messages = &MessageService{client: c}

	return c
}

// Constructing API endpoint. This will returns an *url.URL. Here's the example:
//
//	c := NewClient("1234567", "token", nil)
//	c.EndPoint("Messages", "abcdef") // "/2010-04-01/Accounts/1234567/Messages/abcdef.json"
//
func (c *Client) EndPoint(parts ...string) *url.URL {
	up := []string{apiVersion, "Accounts", c.AccountSid}
	up = append(up, parts...)
	u, _ := url.Parse(strings.Join(up, "/"))
	u.Path = fmt.Sprintf("/%s.%s", u.Path, apiFormat)
	return u
}

func (c *Client) NewRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	ul, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(ul)

	req, _ := http.NewRequest(method, u.String(), body)

	if method == "POST" || method == "PUT" {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	req.SetBasicAuth(c.AccountSid, c.AuthToken)

	req.Header.Add("User-Agent", c.UserAgent)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept-Charset", "utf-8")

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	response := NewResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}

	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	return response, err
}
