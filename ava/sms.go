package main

import (
	"bytes"
	"encoding/xml"
	"os"

	"github.com/subosito/twilio"
)

var tc *twilio.Client

type twilioResp struct {
	XMLName xml.Name `xml:"Response"`
	Message string
}

func bootTwilio() {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	tc = twilio.NewClient(accountSID, authToken, nil)
}

func stringToTwiml(s string) (string, error) {
	var buf bytes.Buffer
	r := &twilioResp{Message: s}
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(r); err != nil {
		return "", err
	}
	return buf.String(), nil
}
