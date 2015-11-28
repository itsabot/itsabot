package task

import (
	"log"
	"strings"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

const (
	addressStateNone float64 = iota
	addressStateAskUser
	addressStateGetName
)

func (t *Task) RequestAddress(dest **dt.Address) (bool, error) {
	t.typ = "Address"
	switch t.getState() {
	case addressStateNone:
		t.resp.Sentence = "Where should I ship it?"
		t.setState(addressStateAskUser)
	case addressStateAskUser:
		addr, remembered, err := language.ExtractAddress(t.ctx.DB,
			t.ctx.Msg.User, t.ctx.Msg.Input.Sentence)
		if err == dt.ErrNoAddress {
			t.resp.Sentence = "I'm sorry. I don't have any record of that place. Where would you like it shipped?"
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if addr == nil || addr.Line1 == "" || addr.City == "" ||
			addr.State == "" {
			t.resp.Sentence = "I'm sorry. I couldn't understand that address. Could you try typing it again more clearly?"
			return false, nil
		}
		addr.Country = "USA"
		var id uint64
		if !remembered {
			log.Println("address was new")
			t.setState(addressStateGetName)
			t.resp.Sentence = "Is that your home or office?"
			id, err = t.ctx.Msg.User.SaveAddress(t.ctx.DB, addr)
			if err != nil {
				return false, err
			}
			log.Println("here... setting interim ID")
			t.setInterimID(id)
			return false, nil
		}
		log.Println("address was not new")
		*dest = addr
		return true, nil
	case addressStateGetName:
		var location string
		tmp := strings.Fields(strings.ToLower(t.ctx.Msg.Input.Sentence))
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
			yes := language.ExtractYesNo(t.ctx.Msg.Input.Sentence)
			if !yes.Bool && yes.Valid {
				return true, nil
			}
		}
		addr, err := t.ctx.Msg.User.UpdateAddressName(t.ctx.DB,
			t.getInterimID(), location)
		if err != nil {
			return false, err
		}
		addr.Name = location
		*dest = addr
		return true, nil
	default:
		log.Println("warn: invalid state", t.getState())
	}
	return false, nil
}
