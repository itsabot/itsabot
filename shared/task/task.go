// Package task defines commonly used tasks across plugins. Plugins can share
// these common tasks and use them in their state machines.
package task

import "github.com/itsabot/abot/shared/datatypes"

// Type references the type of task to perform. Valid options are constant.
type Type int

const (
	// RequestAddress for a given user.
	RequestAddress Type = iota + 1
)

// New returns a slice of States for inclusion into a StateMachine.SetStates()
// call.
func New(p *dt.Plugin, t Type, label string) []dt.State {
	switch t {
	case RequestAddress:
		return getAddress(p, label)
	}
	return []dt.State{}
}
