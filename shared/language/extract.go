package language

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/helpers/address"
)

var regexCurrency = regexp.MustCompile(`\d+\.?\d*`)

// ExtractCurrency returns a pointer to a string to allow a user a simple check
// to see if currency text was found. If the response is nil, no currency was
// found. This API design also maintains consitency when we want to extract and
// return a struct (which should be returned as a pointer).
func ExtractCurrency(s string) *string {
	found := regexCurrency.FindString(s)
	if len(found) == 0 {
		return nil
	}
	found = strings.Replace(found, ".", "", 1)
	return &found
}

func ExtractYesNo(s string) *bool {
	ss := strings.Fields(s)
	for _, w := range ss {
		if yes[w] {
			tru := true
			return &tru
		}
		if no[w] {
			fls := false
			return &fls
		}
	}
	return nil
}

func ExtractAddress(s string) (*datatypes.Address, error) {
	addr, err := address.Parse(s)
	if err != nil {
		return &datatypes.Address{}, err
	}
	type addr2S struct {
		XMLName  xml.Name `xml:"Address"`
		ID       string   `xml:"ID,attr"`
		FirmName string
		Address1 string
		Address2 string
		City     string
		State    string
		Zip5     string
		Zip4     string
	}
	addr2Stmp := addr2S{
		ID:       "0",
		Address1: addr.Line2,
		Address2: addr.Line1,
		City:     addr.City,
		State:    addr.State,
	}
	if len(addr.Zip) > 0 {
		addr2Stmp.Zip5 = addr.Zip[:5]
	}
	if len(addr.Zip) > 5 {
		addr2Stmp.Zip4 = addr.Zip[5:]
	}
	addrS := struct {
		XMLName    xml.Name `xml:"AddressValidateRequest"`
		USPSUserID string   `xml:"USERID,attr"`
		Address    addr2S
	}{
		USPSUserID: os.Getenv("USPS_USER_ID"),
		Address:    addr2Stmp,
	}
	xmlAddr, err := xml.Marshal(addrS)
	if err != nil {
		return &datatypes.Address{}, err
	}
	log.Println(string(xmlAddr))
	u := "https://secure.shippingapis.com/ShippingAPI.dll?API=Verify&XML="
	u += url.QueryEscape(string(xmlAddr))
	response, err := http.Get(u)
	if err != nil {
		return &datatypes.Address{}, err
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return &datatypes.Address{}, err
	}
	resp := struct {
		XMLName    xml.Name `xml:"AddressValidateResponse"`
		USPSUserID string   `xml:"USERID,attr"`
		Address    addr2S
	}{
		USPSUserID: os.Getenv("USPS_USER_ID"),
		Address:    addr2Stmp,
	}
	if err = xml.Unmarshal(contents, &resp); err != nil {
		return &datatypes.Address{}, err
	}
	a := datatypes.Address{
		Name:  resp.Address.FirmName,
		Line1: resp.Address.Address2,
		Line2: resp.Address.Address1,
		City:  resp.Address.City,
		State: resp.Address.State,
	}
	if len(resp.Address.Zip4) > 0 {
		a.Zip = resp.Address.Zip5 + "-" + resp.Address.Zip4
	} else {
		a.Zip = resp.Address.Zip5
	}
	return &a, nil
}
