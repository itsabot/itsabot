// Package driver defines interfaces to be implemented by payment drivers as
// used by package payment.
package driver

// Driver is the interface that must be implemented by a payment driver.
type Driver interface {
	// Open returns a new connection to the payment server. The name is a
	// string in a driver-specific format.
	Open(name string) (Conn, error)
}

// Conn is a connection to the external payment service.
type Conn interface {
	// Charge a customer for something. The isoCurrency is the currency in
	// its 3-letter ISO code.
	Send(amountInCents uint64, isoCurrency string, user *User) error

	// Close the connection.
	Close() error
}
