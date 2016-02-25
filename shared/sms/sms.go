package sms

import (
	"os"
	"regexp"

	"github.com/subosito/twilio"
)

// PhoneRegex determines whether a string is a phone number
var PhoneRegex = regexp.MustCompile(`^\+?[0-9\-\s()]+$`)

// NewClient returns an authorized Twilio client using TWILIO_ACCOUNT_SID and
// TWILIO_AUTH_TOKEN environment variables.
func NewClient() *twilio.Client {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	return twilio.NewClient(accountSID, authToken, nil)
}

// SentMessage sends an SMS using a Twilio client to a specific phone number in
// the following valid international format ("+13105555555") from an owned
// Twilio phone number retrieved from the TWILIO_NUMBER environment variable.
func SendMessage(tc *twilio.Client, to, msg string) error {
	params := twilio.MessageParams{Body: msg}
	_, _, err := tc.Messages.Send(os.Getenv("TWILIO_NUMBER"), to, params)
	return err
}
