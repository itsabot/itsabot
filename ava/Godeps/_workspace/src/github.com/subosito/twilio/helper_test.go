package twilio

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const (
	accountSid = "AC5ef87"
	authToken  = "2ecaf01"
)

var (
	mux    *http.ServeMux
	client *Client
	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = NewClient(accountSid, authToken, nil)
	client.BaseURL, _ = url.Parse(server.URL)
}

func teardown() {
	server.Close()
}

func encodeAuth() string {
	s := accountSid + ":" + authToken
	return ("Basic " + base64.StdEncoding.EncodeToString([]byte(s)))
}

func parseTimestamp(s string) Timestamp {
	tm, _ := time.Parse(time.RFC1123Z, s)
	return Timestamp{Time: tm}
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if want != r.Method {
		t.Errorf("Request method = %v, want %v", r.Method, want)
	}
}
