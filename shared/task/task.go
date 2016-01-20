package task

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/shared/datatypes"
)

type Task struct {
	Done bool
	Err  error

	typ      string
	resultID sql.NullInt64
	sg       *dt.MailClient
	ec       *dt.SearchClient
	tc       *twilio.Client
	db       *sqlx.DB
	msg      *dt.Msg
	respMsg  *dt.RespMsg
}

type Type int
type Opts map[string]string

const (
	RequestAddress Type = iota + 1
	RequestPurchase
)

func Run(db *sqlx.DB, in *dt.Msg, t Type, o Opts) (string, error) {
	str, err := getAddress(db, in)
	return str, err
}

/*
func New(db *sqlx.DB, in *dt.Msg, t Type, o Opts) []*dt.State {
	str, _ := Run(db, in, t, o)

}
*/

/*
func New(db *sqlx.DB, msg *dt.Msg, respMsg *dt.RespMsg) (*Task, error) {
	if msg.State == nil {
		return &Task{}, errors.New("state nil in *dt.Msg")
	}
	return &Task{
		msg:     msg,
		respMsg: respMsg,
		sg:      dt.NewMailClient(),
		ec:      dt.NewSearchClient(),
		tc:      sms.NewClient(),
	}, nil
}
*/

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
	return fmt.Sprintf("__task%s_UserID_%d", t.typ, t.msg.User.ID)
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
