// Package task defines commonly used tasks across plugins. Plugins can share
// these common tasks and use them in their state machines.
package task

import "github.com/itsabot/abot/shared/datatypes"

// Type references the type of task to perform. Valid options are constant.
type Type int

const (
	// RequestAddress for a given user.
	RequestAddress Type = iota + 1

	// RequestCalendar access for a given user.
	RequestCalendar

	// RequestPurchaseAuthZip requests a user's billing zip code to confirm
	// that they're authorized to make a purchase. The request is skipped if
	// the user has authorized by the same or more secure method recently.
	RequestPurchaseAuthZip

	// RequestSignup requests that a user sign up or add their contact
	// information via ABOT_URL/signup.
	RequestSignup
)

// New returns a slice of States for inclusion into a StateMachine.SetStates()
// call.
func New(p *dt.Plugin, t Type, label string) []dt.State {
	switch t {
	case RequestAddress:
		return getAddress(p, label)
	case RequestCalendar:
		return getCalendar(p, label)
	case RequestSignup:
		return requestSignup(p, label)
	}
	return []dt.State{}
}
