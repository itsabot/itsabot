package main

import (
	"os"

	"github.com/subosito/twilio"
)

var tc *twilio.Client

func bootTwilio() {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	tc = twilio.NewClient(accountSID, authToken, nil)
}
