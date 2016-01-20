package dt

import "database/sql"

type Card struct {
	ID             int
	AddressID      sql.NullInt64
	Last4          string
	CardholderName string
	ExpMonth       int
	ExpYear        int
	Brand          string
	StripeID       string
	Zip5Hash       []byte `sql:"zip5hash"`
}
