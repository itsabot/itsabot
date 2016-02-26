// Package driver defines interfaces to be implemented by SMS drivers as used
// by package sms.
package driver

// Driver is the interface that must be implemented by an SMS driver.
type Driver interface {
	// Open returns a new connection to the SMS server. The name is a string
	// in a driver-specific format.
	Open(name string) (Conn, error)

	// SetKeys saves the keys used by an external SMS service in an HTTP
	// request to Abot. These keys are used to retrieve the contents of To,
	// From, and Message from that HTTP request.
	SetKeys(to, from, msg string)
}

// Conn is a connection to the external SMS service.
type Conn interface {
	// Send an SMS from one number to another. It's up to each individual
	// SMS driver to specify the format of the numbers.
	Send(from, to, msg string) error

	// Close the connection.
	Close() error
}

// SMS defines an interface with basic getters to interact with an SMS message.
type SMS interface {
	To() string
	From() string
	Content() string
}
