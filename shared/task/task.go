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
	"github.com/avabot/ava/shared/pkg"
)

type Task struct {
	Done bool
	Err  error

	// TODO simplify all the below into a *dt.Context struct?
	id       uint64
	typ      string
	resultID sql.NullInt64
	db       *sqlx.DB
	u        *dt.User
	resp     *dt.Resp
	respMsg  *dt.RespMsg
	pkg      *pkg.Pkg
}

func New(db *sqlx.DB, u *dt.User, resp *dt.Resp, respMsg *dt.RespMsg,
	pkg *pkg.Pkg) (*Task, error) {
	if resp.State == nil {
		return &Task{}, errors.New("State nil in *dt.Resp")
	}
	if err := p.InitRPCClient(); err != nil {
		return &Task{}, err
	}
	return &Task{
		db:      db,
		u:       u,
		pkg:     pkg,
		resp:    resp,
		respMsg: respMsg,
	}, nil
}

func (t *Task) RequestAddress(dest *dt.Address) (bool, error) {
	table := "addresses"
	q := `
		SELECT id, resultid
		FROM tasks
		WHERE userid=$1 AND resulttable='$2'
		ORDER BY createdat ASC`
	var tmp []struct {
		ID       uint64
		ResultID sql.NullInt64
	}
	err := t.db.Select(&tmp, q, t.u.ID, table)
	if err == sql.ErrNoRows {
		if err = t.save(table); err != nil {
			// kick off the address request process
			t.getAddress()
			return false, err
		}
		return false, nil
	}
	if len(tmp) > 0 {
		for i, taskIDs := range tmp {
			if i == 0 {
				continue
			}
			removeTask(taskIDs)
		}
	}
	if err != nil {
		return false, err
	}
	// not marshaled directly into *Task to keep id and resultID private
	t.id = tmp[0].ID
	t.resultID = tmp[0].ResultID
	if !resID.Valid {
		return false, nil
	}
	if err = t.u.GetAddress(dest, resID); err != nil {
		return false, err
	}
	return true, nil
}

func (t *Task) save(table string) error {
	q := `
		INSERT INTO tasks (userid, packagename, resulttable)
		VALUES ($1, $2, $3, $4)`
	_, err := t.db.Exec(q, t.u.ID, t.pkg, table)
	return err
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

func (t *Task) getAddress() error {
	// TODO add memory of shipping addresses
	t.typ = "Address"
	switch t.getState() {
	case addressStateNone:
		t.resp.Sentence = "Where should I ship it?"
		t.setState(addressStateAskUser)
	case addressStateAskUser:
		t.resp.Sentence = ""
		addr, err := language.ExtractAddress(db, m.Input.Sentence)
		if err != nil {
			return err
		}
		if addr == nil {
			return pkg.SaveResponse(t.respMsg, t.resp)
		}
		if err := t.u.SaveAddress(t.db, addr); err != nil {
			return err
		}
		t.setState(addressStateGetName)
		t.resp.Sentence = "Is that your home or office?"
	case addressStateGetName:
		var location string
		sent := strings.ToLower(m.Input.Sentence)
		for _, w := range sent {
			if w == "home" {
				location = w
				break
			} else if w == "office" || w == "work" {
				location = "office"
				break
			}
		}
		if len(location) == 0 {
			yes := language.ExtractYesNo(db, m.Input.Sentence)
			if !yes.Bool && yes.Valid {
				// send/record this response, then call the pkg
				// again
				t.resp.Sentence = "Got it."
				pkg.SaveResponse(t.respMsg, t.resp)
				// TODO find a way to communicate back to the
				// pkg
				// t.pkg.RPCClient
				return nil
			}
		}
		err := t.u.UpdateAddressName(t.db, getInterimID(), name)
		if err != nil {
			return err
		}
	default:
		log.Println("warn: invalid state", state)
	}
	return pkg.SaveResponse(t.respMsg, t.resp)
}

func (t *Task) getState() float64 {
	return resp.State["__taskState"].(float64)
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
	key := fmt.Sprintf("__task%sID", t.typ)
	switch resp.State[key].(type) {
	case uint64:
		return resp.State[key].(uint64)
	case float64:
		return uint64(resp.State[key].(float64))
	default:
		log.Println("warn: couldn't get interim ID: invalid type",
			reflect.TypeOf(resp.State[key]))
	}
	return uint64(0)
}
