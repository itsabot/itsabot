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
	resp     *dt.Resp
	respMsg  *dt.RespMsg
}

func New(ctx *dt.Ctx, resp *dt.Resp, respMsg *dt.RespMsg) (*Task, error) {
	if resp.State == nil {
		return &Task{}, errors.New("state nil in *dt.Resp")
	}
	return &Task{
		ctx:     ctx,
		resp:    resp,
		respMsg: respMsg,
	}, nil
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
	key := fmt.Sprintf("__task%s_User%dID", t.typ, t.ctx.Msg.User.ID)
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
	key := fmt.Sprintf("__task%s_User%dID", t.typ, t.ctx.Msg.User.ID)
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
