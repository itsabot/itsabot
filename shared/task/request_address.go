package task

import (
	"encoding/json"
	"strings"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/language"
)

func getAddress(p *dt.Plugin, label string) []dt.State {
	return []dt.State{
		{
			Label: label,
			OnEntry: func(in *dt.Msg) string {
				return "Where should I ship to?"
			},
			OnInput: func(in *dt.Msg) {
				addr, mem, err := language.ExtractAddress(p.DB,
					in.User, in.Sentence)
				if addr == nil || err != nil {
					return
				}
				p.SetMemory(in, "shipping_address", addr)
				p.SetMemory(in, "__remembered", mem)
			},
			// TODO consider adding a string to Complete's response
			// and passing in an error from OnInput to customize err
			// responses.
			Complete: func(in *dt.Msg) (bool, string) {
				return p.HasMemory(in, "shipping_address"), ""
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
				mem := p.GetMemory(in, "shipping_address")
				var addr *dt.Address
				err := json.Unmarshal(mem.Val, addr)
				if err != nil {
					return
				}
				addr.Name = location
				p.SetMemory(in, "shipping_address", addr)
			},
			Complete: func(in *dt.Msg) (bool, string) {
				c1 := p.HasMemory(in, "shipping_address")
				c2 := p.HasMemory(in, "__remembered")
				return c1 && c2, ""
			},
		},
	}
}
