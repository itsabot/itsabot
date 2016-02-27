// Package driver defines interfaces to be implemented by SMS drivers as used
// by package sms.
package driver

// Driver is the interface that must be implemented by an SMS driver.
type Driver interface {
	// Open returns a new connection to the SMS server. The name is a string
	// in a driver-specific format.
	Open(name string) (Conn, error)

	// FromKey is the key in an SMS service's request that contains the
	// From telephone number.
	FromKey() string

	// ToKey is the key in an SMS service's request that contains the
	// To telephone number.
	ToKey() string

	// MsgKey is the key in an SMS service's request that contains the
	// content of the message.
	MsgKey() string
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
	// From is the sending phone number.
	From() string

	// To are all of the receiving phone numbers.
	To() []string

	// Content is the body of the message. Currently there is no support for
	// MMS, but we'd like to add it.
	Content() string
}

type PhoneNumber interface {
	Valid() bool
}
