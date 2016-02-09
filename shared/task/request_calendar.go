package task

import "github.com/avabot/ava/shared/datatypes"

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
