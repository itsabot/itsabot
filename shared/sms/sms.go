package sms

import (
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
)

func NewClient() *twilio.Client {
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	return twilio.NewClient(accountSID, authToken, nil)
}

func SendMessage(tc *twilio.Client, to, msg string) error {
	params := twilio.MessageParams{Body: msg}
	_, _, err := tc.Messages.Send("+14242971568", to, params)
	return err
}
