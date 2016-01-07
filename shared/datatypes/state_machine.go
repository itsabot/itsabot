package dt

import (
	"bytes"
	"encoding/gob"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

type StateMachine struct {
	State     int
	Handlers  []State
	Reset     func()
	SetMemory func(string, interface{})
	pkgName   string
	logger    *log.Entry
	db        *sqlx.DB
}

type State struct {
	// OnEntry preprocesses and asks the user for information. If you need
	// to do something when the state begins, like run a search or hit an
	// endpoint, do that within the OnEntry function, since it's only called
	// once.
	OnEntry func() string

	// OnInput sets the category in the cache/DB. Note that if invalid, this
	// state's Complete function will return false, preventing the user from
	// continuing. User messages will continue to hit this OnInput func
	// until Complete returns true.
	//
	// A note on error handling: errors should be logged but are not
	// propogated up to the user. Due to the preferred style of thin
	// States, you should generally avoid logging errors directly in
	// the OnInput function and instead log them within any called functions
	// (e.g. setPreference).
	OnInput func(*Input)

	// Complete will determine if the state machine continues. If true,
	// it'll move to the next state. If false, the user's next response will
	// hit this state's OnInput function again.
	Complete func(*Input) bool

	// Memory will search through preferences about the user. If a past
	// preference is found, it'll skip to the OnInput response, with that
	// preference as the input.
	Memory string
}

func NewStateMachine(ss ...State) *StateMachine {
	sm := StateMachine{State: 0}
	sm.Handlers = ss
	sm.Reset = func() {}
	sm.Logger = log.WithFields(log.Fields{
		"pkg": pkgName,
	})
	return &sm
}

func (sm StateMachine) SetLogger(l *log.Entry) {
	sm.logger = l
}

func (sm StateMachine) SetDBConn(s *sqlx.DB) {
	sm.db = s
}

func (sm StateMachine) SetPkgName(n string) {
	sm.pkgName = n
}

func (sm StateMachine) Next(in *Input) string {
	if sm.State+1 >= len(sm.Handlers) {
		sm.Reset()
		return sm.Handlers[sm.State].OnEntry(in)
	}
	// check completion of current state
	if sm.Handlers[sm.State].Complete(in) {
		sm.State++
		s := sm.Handlers[sm.State].OnEntry(in)
		if len(s) == 0 {
			sm.Logger.WithField("state", sm.State).
				Warnln("OnEntry returned \"\"")
		}
		return s
	}
	// check memory to determine if Ava should skip this state
	mem := sm.Handlers[sm.State].Memory
	if len(mem) > 0 {
		if sm.HasMemory(in, mem) {
			return sm.Next(in)
		}
	}
	sm.Handlers[sm.State].OnInput(in)
	return ""
}

func (sm StateMachine) OnInput(in *Input) {
	sm.Handlers[sm.State].OnInput(in)
}

func (sm StateMachine) SetOnReset(reset func()) {
	sm.Reset = reset
}

func (sm StateMachine) SetMemory(in *Input, k string, v interface{}) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		sm.Logger.Errorln(err, "setting memory at", k, "to", v)
		return
	}
	// the `||` upserts the key into postgres jsonb
	q := `SELECT state FROM states WHERE userid=$1 AND pkgname=$2
	      || jsonb_build_object('%s', '%b')`
	_, err := sm.db.Exec(q, sm.PkgName, in.UserID, k, buf.Bytes())
	if err != nil {
		sm.Logger.Errorln(err, "setting memory at", k, "to", v)
		return
	}
	// TODO set preference here as well
}

func (sm StateMachine) GetMemory(in *Input, k string) Memory {
	q := `SELECT state FROM states WHERE userid=$1 AND pkgname=$2`
	var buf bytes.Buffer
	if err := sm.db.Get(&buf, q, in.UserID, sm.pkgName); err != nil {
		sm.logger.Errorln(err, "getMemory for key", k)
		return []byte{}
	}
	return Memory{Key: k, Val: buf.Bytes(), logger: sm.logger}
}

func (sm StateMachine) HasMemory(in *Input, k string) bool {
	return len(sm.GetMemory(in, k)) > 0
}

func (sm StateMachine) Reset() {
	sm.State = 0
	sm.Reset()
}
