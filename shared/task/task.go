package task

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

type Task struct {
	Done bool
	Err  error

	// TODO simplify all the below into a *dt.Context struct?
	id       uint64
	typ      string
	resultID sql.NullInt64
	db       *sqlx.DB
	msg      *dt.Msg
	resp     *dt.Resp
	respMsg  *dt.RespMsg
}

func New(db *sqlx.DB, msg *dt.Msg, resp *dt.Resp, respMsg *dt.RespMsg) (*Task,
	error) {
	if resp.State == nil {
		return &Task{}, errors.New("state nil in *dt.Resp")
	}
	return &Task{
		db:      db,
		msg:     msg,
		resp:    resp,
		respMsg: respMsg,
	}, nil
}

// Delete removes the task. It's available for packages to call it as well.
// Packages are restricted to one task per user. Packages should handle deleting
// tasks themselves. If a package exceeds that, log a warning and delete the old
// task.
func (t *Task) Delete() error {
	_, err := t.db.Exec(`DELETE FROM tasks WHERE id=$1`, t.id)
	return err
}

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
		addr, remembered, err := language.ExtractAddress(t.db,
			t.msg.User, t.msg.Input.Sentence)
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
			t.setState(addressStateGetName)
			t.resp.Sentence = "Is that your home or office?"
			id, err = t.msg.User.SaveAddress(t.db, addr)
			if err != nil {
				return false, err
			}
			t.setInterimID(id)
			return false, nil
		}
		*dest = addr
		return true, nil
	case addressStateGetName:
		var location string
		tmp := strings.Fields(strings.ToLower(t.msg.Input.Sentence))
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
			yes := language.ExtractYesNo(t.msg.Input.Sentence)
			if !yes.Bool && yes.Valid {
				return true, nil
			}
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
		log.Println("warn: invalid state", t.getState())
	}
	return false, nil
}

func (t *Task) getState() float64 {
	tmp := t.resp.State["__taskState"]
	if tmp == nil {
		return addressStateNone
	}
	return tmp.(float64)
}

func (t *Task) setState(s float64) {
	t.resp.State["__taskState"] = s
}

func (t *Task) setInterimID(id uint64) {
	key := fmt.Sprintf("__task%s_User%dID", t.typ, t.msg.User.ID)
	t.resp.State[key] = id
}

// getInterimID is useful when you've saved an object, but haven't finished
// modifying it, yet. For example, addresses are saved, but named after the
// fact. If we save the resultID into the task table, the task will cede control
// back to its calling package. As a result, we save the interimID in the resp
// state to keep task control.
func (t *Task) getInterimID() uint64 {
	if len(t.typ) == 0 {
		log.Println("warn: t.typ should be set but was \"\"")
	}
	key := fmt.Sprintf("__task%s_User%dID", t.typ, t.msg.User.ID)
	switch t.resp.State[key].(type) {
	case uint64:
		return t.resp.State[key].(uint64)
	case float64:
		return uint64(t.resp.State[key].(float64))
	default:
		log.Println("warn: couldn't get interim ID: invalid type",
			reflect.TypeOf(t.resp.State[key]))
	}
	return uint64(0)
}
