// Package driver defines interfaces to be implemented by SMS drivers as used
// by package sms.
package driver

import "github.com/julienschmidt/httprouter"

// Driver is the interface that must be implemented by an SMS driver.
type Driver interface {
	// Open returns a new connection to the SMS server. Authentication is
	// handled by the individual drivers.
	Open(r *httprouter.Router) (Conn, error)
}

// Conn is a connection to the external SMS service.
type Conn interface {
	// Send an SMS from one number to another. It's up to each individual
	// SMS driver to specify the format of the numbers. The From number is
	// handled by the drivers themselves.
	Send(to, msg string) error

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

// PhoneNumber defines an interface by which phone numbers can be validated.
type PhoneNumber interface {
	// Valid determines if a specific phone number is valid for a given
	// SMS driver.
	Valid() bool
}
