// Package driver defines interfaces to be implemented by email drivers as used
// by package email.
package driver

import (
	"time"

	"github.com/itsabot/abot/shared/datatypes"
)

// Driver is the interface that must be implemented by an email driver.
type Driver interface {
	// Open returns a new connection to the email server. The name is a
	// string in a driver-specific format, often for authentication.
	Open(name string) (Conn, error)
}

// Conn is a connection to the external email service.
type Conn interface {
	// GetEmails returns emails with a given time range. Further searching
	// should be done on the retrieved emails.
	GetEmails(dt.TimeRange) ([]Email, error)

	// Close the connection.
	Close() error
}

// Email represents a single event in a email.
type Email interface {
	// From is the email address that sent the message.
	From() string

	// To is the set of email addresses that the message was sent to.
	To() []string

	// Title of the event
	Subject() string

	// Body content of the email
	Body() string

	// BodyType of the email (HTML, plaintext, etc.)
	BodyType() string

	// SentAt is the time the email was sent.
	SentAt() *time.Time

	// Send the email. Return an error if the server rejects the message.
	Send() error
}
