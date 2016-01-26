package task

import (
	"encoding/json"
	"strings"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

const (
	addressStateNone float64 = iota
	addressStateAskUser
	addressStateGetName
)

func getAddress(sm *dt.StateMachine) []dt.State {
	db := sm.GetDBConn()
	return []dt.State{
		{
			OnEntry: func(in *dt.Msg) string {
				return "Ok. Where should I ship this?"
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
			Memory: "__remembered",
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
				return sm.HasMemory(in, "shipping_address"), ""
			},
		},
	}
}
