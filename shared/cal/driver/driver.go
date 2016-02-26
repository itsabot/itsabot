// Package driver defines interfaces to be implemented by calendar drivers as
// used by package cal.
package driver

import (
	"time"

	"github.com/jmoiron/sqlx"
)

// Driver is the interface that must be implemented by a calendar driver.
type Driver interface {
	// Open returns a new connection to the calendar server. The name is a
	// string in a driver-specific format. A database connection is passed
	// in to enable the driver to retrieve existing auth tokens.
	Open(db *sqlx.DB, name string) (Conn, error)
}

// Conn is a connection to the external calendar service.
type Conn interface {
	// GetEvents returns events with a given time range. Further searching
	// should be done on the retrieved events.
	GetEvents(TimeRange) ([]Event, error)

	// Close the connection.
	Close() error
}

// Event represents a single event in a calendar.
type Event interface {
	// Title of the event
	Title() string

	// Location of the event in a free-form string
	Location() string

	// StartTime of the event
	StartTime() *time.Time

	// DurationInMins of the event. This is used rather than an endtime to
	// keep client implementations simple and prevent mixing timezones
	// between start and end dates.
	DurationInMins() int

	// Recurring specifies if the event happens more than once.
	Recurring() bool

	// RecurringFreq specifies how often the event occurs.
	RecurringFreq() RecurringFreq

	// AllDay specifies whether the event is running all day.
	AllDay() bool

	// Attendees of the event
	Attendees() []*Attendee

	// Create the event on the remote server.
	Create() error

	// Update the event on the remote server.
	Update() error
}

// Attendee of an Event
type Attendee interface {
	// Name of the attendee
	Name() string

	// Email of the attendee
	Email() string

	// Phone number of the attendee
	Phone() string
}

// TimeRange defines a range of time in searching for events or creating an
// event for a specified duration.
type TimeRange struct {
	Start *time.Time
	End   *time.Time
}

// RecurringFreq specifies how often an event recurs.
type RecurringFreq int

// Define options for event recurring frequencies.
const (
	RecurringFreqOnce RecurringFreq = iota
	RecurringFreqDaily
	RecurringFreqWeekly
	RecurringFreqMonthly
	RecurringFreqYearly
)
