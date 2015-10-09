package twilio

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient(accountSid, authToken, nil)
	assert.Equal(t, c.BaseURL.String(), apiBaseURL)
	assert.Equal(t, c.UserAgent, userAgent)
}

func TestNewRequest(t *testing.T) {
	c := NewClient(accountSid, authToken, nil)

	inURL := "/foo"
	outURL := c.BaseURL.String() + "/foo"

	req, _ := c.NewRequest("GET", inURL, nil)

	userAgent := req.Header.Get("User-Agent")
	assert.Equal(t, userAgent, c.UserAgent)
	assert.Equal(t, req.URL.String(), outURL)
	assert.Equal(t, req.Header.Get("Authorization"), encodeAuth())
}

func TestNewRequest_badURL(t *testing.T) {
	c := NewClient(accountSid, authToken, nil)

	_, err := c.NewRequest("GET", ":", nil)
	assert.NotNil(t, err)

	erx, ok := err.(*url.Error)
	assert.True(t, ok)
	assert.NotNil(t, erx)
	assert.Equal(t, erx.Op, "parse")
}

func TestDo(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		Bar string
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, want %v", r.Method, m)
		}

		fmt.Fprint(w, `{"Bar":"bar"}`)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	body := new(foo)
	client.Do(req, body)

	want := &foo{"bar"}
	assert.Equal(t, body, want)
}

func TestDo_httpError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(req, nil)
	assert.NotNil(t, err)
}

func TestDo_redirectLoop(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(req, nil)
	assert.NotNil(t, err)

	err, ok := err.(*url.Error)
	assert.True(t, ok)
	assert.NotNil(t, err)
}

func TestEndPoint(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Hello", "123")
	want, _ := url.Parse("/2010-04-01/Accounts/AC5ef87/Hello/123.json")
	assert.Equal(t, u, want)
}
