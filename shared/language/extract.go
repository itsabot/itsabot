package language

import (
	"encoding/xml"
	"errors"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/helpers/address"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var regexCurrency = regexp.MustCompile(`\d+\.?\d*`)
var regexNum = regexp.MustCompile(`\d+`)
var regexNonWords = regexp.MustCompile(`[^\w\s]`)

// ErrNotFound is thrown when the requested type cannot be found in the string
var ErrNotFound error = errors.New("couldn't extract requested type from string")

// ExtractCurrency returns an int64 if a currency is found, and throws an
// error if one isn't.
func ExtractCurrency(s string) (int64, error) {
	log.Debug("extracting currency")
	s = regexCurrency.FindString(s)
	if len(s) == 0 {
		return 0, ErrNotFound
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	log.Debug("found value", val)
	// convert parsed float into an int64 with precision of 2 decimal places
	return int64(val * 100), nil
}

// ExtractYesNo determines whether a string contains a Yes or No response.
// This is useful for plugins to determine a user's answer to a Yes/No question.
func ExtractYesNo(s string) (bool, error) {
	ss := strings.Fields(strings.ToLower(s))
	for _, w := range ss {
		w = strings.TrimRight(w, " .,;:!?'\"")
		if yes[w] {
			return true, nil
		}
		if no[w] {
			return false, nil
		}
	}
	return false, ErrNotFound
}

// ExtractAddress will return an address from a user's message, whether it's a
// labeled address (e.g. "home", "office"), or a full U.S. address (e.g. 100
// Penn St., CA 90000)
func ExtractAddress(db *sqlx.DB, u *dt.User, s string) (*dt.Address, bool, error) {
	addr, err := address.Parse(s)
	if err != nil {
		// check if user's address is in DB already
		log.Debug("Checking if user address already in DB...")
		if addr, err = u.GetAddress(db, s); err == nil {
			return addr, true, nil
		}
		return nil, false, err
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
		Zip5:     addr.Zip5,
		Zip4:     addr.Zip4,
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
		return nil, false, err
	}
	log.Debug(string(xmlAddr))
	ul := "https://secure.shippingapis.com/ShippingAPI.dll?API=Verify&XML="
	ul += url.QueryEscape(string(xmlAddr))
	response, err := http.Get(ul)
	if err != nil {
		return nil, false, err
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, false, err
	}
	if err = response.Body.Close(); err != nil {
		return nil, false, err
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
		log.Debug("USPS response", string(contents))
		return nil, false, err
	}
	a := dt.Address{
		Name:  resp.Address.FirmName,
		Line1: resp.Address.Address2,
		Line2: resp.Address.Address1,
		City:  resp.Address.City,
		State: resp.Address.State,
		Zip5:  resp.Address.Zip5,
		Zip4:  resp.Address.Zip4,
	}
	if len(resp.Address.Zip4) > 0 {
		a.Zip = resp.Address.Zip5 + "-" + resp.Address.Zip4
	} else {
		a.Zip = resp.Address.Zip5
	}
	return &a, false, nil
}

// ExtractCount returns a number from a user's message, useful in situations
// like:
//	Ava>  How many would you like to buy?
//	User> Order 5
func ExtractCount(s string) (int64, error) {
	s = regexNum.FindString(s)
	if len(s) == 0 {
		return 0, ErrNotFound
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// ExtractCities efficiently from a user's message.
func ExtractCities(db *sqlx.DB, in *dt.Msg) ([]dt.City, error) {
	// Interface type is used to expand the args in db.Select below.
	// Although we're only storing strings, []string{} doesn't work.
	var args []interface{}

	// Look for "at", "in", "on" prepositions to signal that locations
	// follow, skipping everything before
	var start int
	for i := range in.Stems {
		switch in.Stems[i] {
		case "at", "in", "on":
			start = i
			break
		}
	}

	// Prepare sentence for iteration
	tmp := regexNonWords.ReplaceAllString(in.Sentence, "")
	words := strings.Fields(tmp)

	// Iterate through words and bigrams to assemble a DB query
	for i := start; i < len(words); i++ {
		args = append(args, words[i])
	}
	bgs := bigrams(words, start)
	for i := 0; i < len(bgs); i++ {
		args = append(args, bgs[i])
	}

	cities := []dt.City{}
	q := `SELECT name, countrycode FROM cities WHERE countrycode='US' AND name IN (?) ORDER BY LENGTH(name) DESC`
	query, arguments, err := sqlx.In(q, args)
	query = db.Rebind(query)
	rows, err := db.Query(query, arguments...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		city := dt.City{}
		if err = rows.Scan(&city.Name, &city.CountryCode); err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	return cities, nil
}

func bigrams(words []string, startIndex int) (bigrams []string) {
	for i := startIndex; i < len(words)-1; i++ {
		bigrams = append(bigrams, words[i]+" "+words[i+1])
	}
	return bigrams
}
