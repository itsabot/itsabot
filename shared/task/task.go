package task

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/avabot/ava/shared/datatypes"
)

type Task struct {
	Done bool
	Err  error

	typ      string
	resultID sql.NullInt64
	ctx      *dt.Ctx
	msg      *dt.Msg
	respMsg  *dt.RespMsg
}

func New(ctx *dt.Ctx, msg *dt.Msg, respMsg *dt.RespMsg) (*Task, error) {
	if msg.State == nil {
		return &Task{}, errors.New("state nil in *dt.Msg")
	}
	return &Task{
		ctx:     ctx,
		msg:     msg,
		respMsg: respMsg,
	}, nil
}

func (t *Task) GetState() float64 {
	tmp := t.msg.State[t.Key()]
	if tmp == nil {
		return addressStateNone
	}
	switch tmp.(type) {
	case float64:
		return tmp.(float64)
	case uint64:
		return float64(tmp.(uint64))
	}
	log.Println("err: state was type", reflect.TypeOf(tmp))
	return 0.0
}

func (t *Task) setState(s float64) {
	t.msg.State[t.Key()] = s
}

func (t *Task) ResetState() {
	t.msg.State[t.Key()] = 0.0
}

func (t *Task) setInterimID(id uint64) {
	t.msg.State[t.Key()] = id
}

func (t *Task) Key() string {
	return fmt.Sprintf("__task%s_UserID_%d", t.typ, t.ctx.Msg.User.ID)
}

// getInterimID is useful when you've saved an object, but haven't finished
// modifying it, yet. For example, addresses are saved, but named after the
// fact. If we save the resultID into the task table, the task will cede control
// back to its calling package. As a result, we save the interimID in the msg
// state to keep task control.
func (t *Task) getInterimID() uint64 {
	if len(t.typ) == 0 {
		log.Println("warn: t.typ should be set but was \"\"")
	}
	switch t.msg.State[t.Key()].(type) {
	case uint64:
		return t.msg.State[t.Key()].(uint64)
	case float64:
		return uint64(t.msg.State[t.Key()].(float64))
	default:
		log.Println("warn: couldn't get interim ID: invalid type",
			reflect.TypeOf(t.msg.State[t.Key()]))
	}
	return uint64(0)
}
