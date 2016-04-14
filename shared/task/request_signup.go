package task

import (
	"fmt"
	"os"

	"github.com/itsabot/abot/shared/datatypes"
)

// requestSignup asks the user to sign up or login via the ABOT_URL/signup.
func requestSignup(sm *dt.StateMachine, label string) []dt.State {
	return []dt.State{
		{
			OnEntry: func(in *dt.Msg) string {
				s := os.Getenv("ABOT_URL")
				return fmt.Sprintf("Please sign up or associate this contact information with your account to continue: %s/signup", s)
			},
			OnInput: func(in *dt.Msg) {
			},
			Complete: func(in *dt.Msg) (bool, string) {
				return in.User.Registered(), ""
			},
		},
	}
}
