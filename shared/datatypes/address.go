package dt

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

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

var ErrNoAddress = errors.New("no address")

// GetAddress searches the database for a specific address by its ID.
func GetAddress(dest *Address, db *sqlx.DB, id uint64) error {
	q := `SELECT id, line1, line2, city, state, country, zip WHERE id=$1`
	return db.Get(dest, q, id)
}
