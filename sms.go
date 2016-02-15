package main

import (
	"bytes"
	"encoding/xml"
)

// twilioResp is an XML struct constructed as a response to Twilio's API to
// respond to user messages via SMS.
type twilioResp struct {
	XMLName xml.Name `xml:"Response"`
	Message string
}

// stringToTwiml converts a string, such as a message to send to a user, into an
// XML struct that Twilio can understand.
func stringToTwiml(s string) (string, error) {
	var buf bytes.Buffer
	r := &twilioResp{Message: s}
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(r); err != nil {
		return "", err
	}
	return buf.String(), nil
}
