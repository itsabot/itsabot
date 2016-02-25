package task

import (
	"encoding/json"
	"strings"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/language"
)

func getAddress(sm *dt.StateMachine, label string) []dt.State {
	db := sm.GetDBConn()
	return []dt.State{
		{
			Label: label,
			OnEntry: func(in *dt.Msg) string {
				return "Where should I ship to?"
			},
			OnInput: func(in *dt.Msg) {
				addr, mem, err := language.ExtractAddress(db,
					in.User, in.Sentence)
				if addr == nil || err != nil {
					return
				}
				sm.SetMemory(in, "shipping_address", addr)
				sm.SetMemory(in, "__remembered", mem)
			},
			// TODO consider adding a string to Complete's response
			// and passing in an error from OnInput to customize err
			// responses.
			Complete: func(in *dt.Msg) (bool, string) {
				return sm.HasMemory(in, "shipping_address"), ""
			},
		},
		{
			SkipIfComplete: true,
			OnEntry: func(in *dt.Msg) string {
				return "Is that your home or office?"
			},
			// TODO consider returning an error message here...
			OnInput: func(in *dt.Msg) {
				var location string
				tmp := strings.Fields(strings.ToLower(
					in.Sentence))
				for _, w := range tmp {
					if w == "home" {
						location = w
						break
					} else if w == "office" || w == "work" {
						location = "office"
						break
					}
				}
				mem := sm.GetMemory(in, "shipping_address")
				var addr *dt.Address
				err := json.Unmarshal(mem.Val, addr)
				if err != nil {
					return
				}
				addr.Name = location
				sm.SetMemory(in, "shipping_address", addr)
			},
			Complete: func(in *dt.Msg) (bool, string) {
				c1 := sm.HasMemory(in, "shipping_address")
				c2 := sm.HasMemory(in, "__remembered")
				return c1 && c2, ""
			},
		},
	}
}
