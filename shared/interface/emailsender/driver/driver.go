// Package driver defines interfaces to be implemented by email_sender drivers
// as used by package emailsender.
package driver

// Driver is the interface that must be implemented by an email driver.
type Driver interface {
	// Open returns a new connection to the email server. The name is a
	// string in a driver-specific format, often for authentication.
	Open(name string) (Conn, error)
}

// Conn is a connection to the external email service.
type Conn interface {
	// SendHTML sends an HTML email to multiple recipients.
	SendHTML(to []string, from, subj, html string) error

	// SendPlainText sends a PlainText email to multiple recipients.
	SendPlainText(to []string, from, subj, plaintext string) error

	// Close the connection.
	Close() error
}
