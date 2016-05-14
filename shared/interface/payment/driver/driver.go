// Package driver defines interfaces to be implemented by payment drivers as
// used by package payment.
package driver

import (
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
)

// Driver is the interface that must be implemented by a payment driver.
type Driver interface {
	// Open returns a new connection to the payment server. The echo router
	// is included to drivers to extend routes at runtime. The name is a
	// string in a driver-specific format.
	Open(db *sqlx.DB, r *httprouter.Router, name string) (Conn, error)
}

// Conn is a connection to the external payment service.
type Conn interface {
	// SaveCard saves limited information about the card to the Cards table
	// in the database including a hashed zipcode and a payment service card
	// token. Returns the ID of the newly created card from the database.
	SaveCard(params *dt.CardParams, user *dt.User) (cardID uint64, err error)

	// Charge a customer for something. The isoCurrency is the currency in
	// its 3-letter ISO code.
	ChargeCard(cardID uint64, amountInCents uint64, isoCurrency string) error

	// RegisterUser on the external payment service.
	RegisterUser(user *dt.User) error

	// Close the connection.
	Close() error
}
