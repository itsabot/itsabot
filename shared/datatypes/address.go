package dt

import "errors"

// Address holds all relevant information in an address for presentation to the
// user and communication to external services, including the USPS address
// validation tool. Right now, an effort is only made to support US-based
// addresses.
type Address struct {
	ID             uint64
	Name           string
	Line1          string
	Line2          string
	City           string
	State          string
	Zip            string
	Zip5           string
	Zip4           string
	Country        string
	DisplayAddress string
}

// ErrNoAddress signals that no address could be found when one was expected.
var ErrNoAddress = errors.New("no address")
