package dt

import "time"

// Method allows you as the package developer to control the level of security
// required in an authentication. Select an appropriate security level depending
// upon your risk tolerance for fraud compared against the quality and ease of
// the user experience.
type Method int

type Authorization struct {
	ID           int
	UserID       int
	Attempts     int
	AuthMethod   Method
	CreatedAt    *time.Time
	AuthorizedAt *time.Time
}
