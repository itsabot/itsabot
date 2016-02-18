package dt

import "time"

// Location represents some location saved for a user or package. This is used
// by itsabot.org/abot/shared/knowledge to quickly retrieve either the user's last location (if
// recent) or request another location using the previous as a hint, e.g. "Are
// you still in Los Angeles?"
type Location struct {
	Name      string
	Lat       float64
	Lon       float64
	CreatedAt time.Time
}

// IsRecent is a helper function to determine if the user's location was last
// recorded in the past day. Beyond that, itsabot.org/abot/shared/knowledge will request an
// updated location.
func (l Location) IsRecent() bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return l.CreatedAt.After(yesterday)
}
