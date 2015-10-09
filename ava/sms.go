package main

import (
	"os"

	"github.com/subosito/twilio"
)

var tc *twilio.Client

// TwilioMsg represents an inbound or outbound SMS or MMS. More types are
// available and documented here:
// https://www.twilio.com/docs/api/twiml/sms/twilio_request
type TwilioMsg struct {
	From             string
	To               string
	Body             string
	MessageSID       string `json:"MessageSid"`
	AccountSID       string `json:"AccountSid"`
	MediaURL         string `json:"MediaUrl"`
	FromCity         string
	FromZip          string
	FromCountry      string
	MediaContentType string
	NumMedia         string
}

func bootTwilio() {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	tc = twilio.NewClient(accountSID, authToken, nil)
}
