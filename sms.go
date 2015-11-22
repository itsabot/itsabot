package main

import (
	"bytes"
	"encoding/xml"
)

type twilioResp struct {
	XMLName xml.Name `xml:"Response"`
	Message string
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
