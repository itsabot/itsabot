package task

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

const (
	addressStateNone float64 = iota
	addressStateAskUser
	addressStateGetName
)

func getAddress(db *sqlx.DB, in *dt.Msg) (string, error) {
	sm, err := dt.NewStateMachine("task_address")
	sm.SetStates(
		dt.State{
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
			Complete: func(in *dt.Msg) bool {
				return sm.HasMemory(in, "shipping_address")
			},
		},
		dt.State{
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
			Complete: func(in *dt.Msg) bool {
				return sm.HasMemory(in, "shipping_address")
			},
		},
	)
	if err != nil {
		return "", err
	}
	return sm.Next(in), nil
}

func (t *Task) RequestAddress(dest **dt.Address, prodCount int) (bool, error) {
	t.typ = "Address"
	done, err := t.getAddress(dest, prodCount)
	if done {
		t.setState(addressStateNone)
	}
	return done, err
}

func (t *Task) getAddress(dest **dt.Address, prodCount int) (bool, error) {
	var pro string
	if prodCount == 1 {
		pro = "it"
	} else {
		pro = "them"
	}
	switch t.GetState() {
	case addressStateNone:
		t.msg.Sentence = "Ok. Where should I ship " + pro + "?"
		t.setState(addressStateAskUser)
	case addressStateAskUser:
		addr, remembered, err := language.ExtractAddress(t.db,
			t.msg.User, t.msg.Sentence)
		if err == dt.ErrNoAddress {
			t.msg.Sentence = "I'm sorry. I don't have any record of that place. Where would you like " + pro + " shipped?"
			return false, nil
		}
		if err != nil {
			t.msg.Sentence = "I'm sorry, but something went wrong. Please try sending that to me again later."
			return false, err
		}
		if addr == nil || addr.Line1 == "" || addr.City == "" ||
			addr.State == "" {
			t.msg.Sentence = "I'm sorry. I couldn't understand that address. Could you try typing it in this format? 1400 Evergreen Ave, Apt 200, Los Angeles, CA"
			return false, nil
		}
		addr.Country = "USA"
		var id uint64
		if !remembered {
			log.Println("address was new")
			t.msg.Sentence = "Is that your home or office?"
			id, err = t.msg.User.SaveAddress(t.db, addr)
			if err != nil {
				return false, err
			}
			log.Println("here... setting interim ID")
			t.setInterimID(id)
			t.setState(addressStateGetName)
			log.Println("set state to get name", addressStateGetName)
			return false, nil
		}
		log.Println("address was not new")
		*dest = addr
		return true, nil
	case addressStateGetName:
		var location string
		tmp := strings.Fields(strings.ToLower(t.msg.Sentence))
		for _, w := range tmp {
			if w == "home" {
				location = w
				break
			} else if w == "office" || w == "work" {
				location = "office"
				break
			}
		}
		if len(location) == 0 {
			return true, nil
		}
		addr, err := t.msg.User.UpdateAddressName(t.db,
			t.getInterimID(), location)
		if err != nil {
			return false, err
		}
		addr.Name = location
		*dest = addr
		return true, nil
	default:
		log.Println("warn: invalid state", t.GetState())
	}
	return false, nil
}
