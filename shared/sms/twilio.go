package sms

import (
	"bytes"
	"encoding/xml"
)

// TwilioResp is an XML struct constructed as a response to Twilio's API to
// respond to user messages via SMS.
type TwilioResp struct {
	XMLName xml.Name `xml:"Response"`
	Message string
}

// StringToTwiml converts a string, such as a message to send to a user, into an
// XML struct that Twilio can understand.
func StringToTwiml(s string) (string, error) {
	var buf bytes.Buffer
	r := &TwilioResp{Message: s}
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(r); err != nil {
		return "", err
	}
	return buf.String(), nil
}
