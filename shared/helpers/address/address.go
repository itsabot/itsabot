package address

import (
	"errors"
	"regexp"
	"strings"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/log"
)

var ErrInvalidAddress = errors.New("invalid address")

var regexAddress = regexp.MustCompile(
	`\d+\s+[a-zA-Z#-'\s\.\,\n\d]*(\d{5}-\d{4}|\d{5})?`)

// regexStreet is useful to search within a regexAddress substring match
var regexStreet = regexp.MustCompile(`^\d+\s+[\w#-'\s\.\n]*$`)

// regexApartment is useful to search within a regexAddress substring match
// after the city has been removed.
var regexApartment = regexp.MustCompile(`(,\s*)?[#\s\.\w]*[\w\s]+$`)

// regexCity is useful to search within a regexAddress substring match after the
// state has been removed.
var regexCity = regexp.MustCompile(`(,\s*)?([a-zA-Z]{2}|\s\w+\s*\w*)$`)

// regexState is useful to search within a regexAddress substring match after
// the zip code has been removed
var regexState = regexp.MustCompile(`(,\s*)?([a-zA-Z]{2}|\s\w+\s*\w*)(,\s*)?$`)

// regexZip is useful to search within a regexAddress substring match
var regexZip = regexp.MustCompile(`(\d{5}-\d{4}|\d{5})$`)

var states = map[string]string{
	"alabama":        "AL",
	"alaska":         "AK",
	"arizona":        "AZ",
	"arkansas":       "AR",
	"california":     "CA",
	"colorado":       "CO",
	"connecticut":    "CT",
	"delaware":       "DE",
	"florida":        "FL",
	"georgia":        "GA",
	"hawaii":         "HI",
	"idaho":          "ID",
	"illinois":       "IL",
	"indiana":        "IN",
	"iowa":           "IA",
	"kansas":         "KS",
	"kentucky":       "KY",
	"lousiana":       "LA",
	"maine":          "ME",
	"maryland":       "MD",
	"massachusetts":  "MA",
	"michigan":       "MI",
	"minnesota":      "MN",
	"mississippi":    "MS",
	"missouri":       "MO",
	"montana":        "MT",
	"nebraska":       "NE",
	"nevada":         "NV",
	"new hampshire":  "NH",
	"new jersey":     "NJ",
	"new mexico":     "NM",
	"new york":       "NY",
	"north carolina": "NC",
	"n carolina":     "NC",
	"north dakota":   "ND",
	"n dakota":       "ND",
	"ohio":           "OH",
	"oklahoma":       "OK",
	"oregon":         "OR",
	"pennsylvania":   "PA",
	"rhode island":   "RI",
	"s carolina":     "SC",
	"south carolina": "SC",
	"s dakota":       "SD",
	"south dakota":   "SD",
	"tennessee":      "TN",
	"texas":          "TX",
	"utah":           "UT",
	"vermont":        "VT",
	"virginia":       "VA",
	"washington":     "WA",
	"w virginia":     "WV",
	"west virginia":  "WV",
	"wisconsin":      "WI",
	"wyoming":        "WY",
}

// Parse a string to return a fully-validated U.S. address.
func Parse(s string) (*dt.Address, error) {
	s = regexAddress.FindString(s)
	if len(s) == 0 {
		log.Debug("missing address")
		return nil, ErrInvalidAddress
	}
	log.Debug("address", s)
	tmp := regexZip.FindStringIndex(s)
	var zip string
	if tmp != nil {
		zip = s[tmp[0]:tmp[1]]
		s = s[:tmp[0]]
	} else {
		log.Debug("no zip found")
	}
	tmp2 := regexState.FindStringIndex(s)
	if tmp2 == nil && tmp == nil {
		log.Debug("no state found AND no zip found")
		return &dt.Address{}, ErrInvalidAddress
	}
	var city, state string
	if tmp2 != nil {
		state = s[tmp2[0]:tmp2[1]]
		s = s[:tmp2[0]]
		state = strings.Trim(state, ", \n")
		if len(state) > 2 {
			state = strings.ToLower(state)
			state = states[state]
		}
		tmp = regexCity.FindStringIndex(s)
		if tmp == nil {
			log.Debug("no city found")
			return &dt.Address{}, ErrInvalidAddress
		}
		city = s[tmp[0]:tmp[1]]
		s = s[:tmp[0]]
	} else {
		log.Debug("no state found")
	}
	tmp = regexApartment.FindStringIndex(s)
	var apartment string
	if tmp != nil {
		apartment = s[tmp[0]:tmp[1]]
		s2 := s[:tmp[0]]
		if len(s2) == 0 {
			apartment = ""
		} else {
			s = s2
		}
	} else {
		log.Debug("no apartment found")
	}
	tmp = regexStreet.FindStringIndex(s)
	if tmp == nil {
		log.Debug(s)
		log.Debug("no street found")
		return &dt.Address{}, ErrInvalidAddress
	}
	street := s[tmp[0]:tmp[1]]
	return &dt.Address{
		Line1:   strings.Trim(street, " \n,"),
		Line2:   strings.Trim(apartment, " \n,"),
		City:    strings.Trim(city, " \n,"),
		State:   strings.Trim(state, " \n,"),
		Zip:     strings.Trim(zip, " \n,"),
		Country: "USA",
	}, nil
}
