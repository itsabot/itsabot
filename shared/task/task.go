package task

import (
	"database/sql"

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
}

type Type int

const (
	RequestAddress Type = iota + 1
	RequestCalendar
	RequestPurchaseAuthZip
)

func New(sm *dt.StateMachine, t Type, label string) []dt.State {
	switch t {
	case RequestAddress:
		return getAddress(sm, label)
	case RequestCalendar:
		return getCalendar(sm, label)
	}
	return []dt.State{}
}
