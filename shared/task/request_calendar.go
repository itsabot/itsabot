package task

import "github.com/itsabot/abot/shared/datatypes"

// getCalendar asks the user to connect their Google calendar via a secure web
// interface (user profile). Eventually this will support adding additional
// calendar services beyond just Google's.
func getCalendar(sm *dt.StateMachine, label string) []dt.State {
	return []dt.State{
		{
			OnEntry: func(in *dt.Msg) string {
				return "You can connect your Google calendar on your profile here: https://avabot.co/?/profile"
			},
			OnInput: func(in *dt.Msg) {
			},
			Complete: func(in *dt.Msg) (bool, string) {
				return true, ""
			},
		},
	}
}
