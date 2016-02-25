package task

import "github.com/itsabot/abot/shared/datatypes"

// Type references the type of task to perform. Valid options are constant
type Type int

const (
	RequestAddress Type = iota + 1
	RequestCalendar
	RequestPurchaseAuthZip
)

// New returns a slice of States for inclusion into a StateMachine.SetStates()
// call.
func New(sm *dt.StateMachine, t Type, label string) []dt.State {
	switch t {
	case RequestAddress:
		return getAddress(sm, label)
	case RequestCalendar:
		return getCalendar(sm, label)
	}
	return []dt.State{}
}
