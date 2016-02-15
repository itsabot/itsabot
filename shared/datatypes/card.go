package dt

import "database/sql"

// Card represents a credit card. Note that information such as the card number,
// security code and zip code are not present in this struct, since that data
// should never hit the server and thus can never be stored. Storage of that
// sensitive data is outsourced to a payment provider (initially Stripe) and is
// sent directly from the client to that payment provider, bypassing the server
// entirely.
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
