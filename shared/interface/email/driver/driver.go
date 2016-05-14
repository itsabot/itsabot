// Package driver defines interfaces to be implemented by email drivers
// as used by package email.
package driver

import (
	"github.com/julienschmidt/httprouter"
)

// Driver is the interface that must be implemented by an email driver.
type Driver interface {
	// Open returns a new connection to the email server. Authentication is
	// handled by the individual drivers.
	Open(r *httprouter.Router) (Conn, error)
}

// Conn is a connection to the external email service.
type Conn interface {
	// SendHTML email through the opened driver connection.
	SendHTML(to []string, from, subj, html string) error

	// SendPlainText email through the opened driver connection.
	SendPlainText(to []string, from, subj, plaintext string) error

	// Close the connection.
	Close() error
}

// Email defines an interface with basic getters to interact with an email message.
type Email interface {
	// From is the sending email address.
	From() string

	// Content is the body of the message.
	Content() string
}
