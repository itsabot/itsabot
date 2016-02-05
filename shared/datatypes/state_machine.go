package dt

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"strconv"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

const stateKey string = "__state"
const stateEnteredKey string = "__state_entered"

type StateMachine struct {
	Handlers     []State
	state        int
	stateEntered bool
	states       map[string]int
	keys         []string
	pkgName      string
	logger       *log.Entry
	db           *sqlx.DB
	resetFn      func(*Msg)
}

type State struct {
	// OnEntry preprocesses and asks the user for information. If you need
	// to do something when the state begins, like run a search or hit an
	// endpoint, do that within the OnEntry function, since it's only called
	// once.
	OnEntry func(*Msg) string

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
	OnInput func(*Msg)

	// Complete will determine if the state machine continues. If true,
	// it'll move to the next state. If false, the user's next response will
	// hit this state's OnInput function again.
	Complete func(*Msg) (bool, string)

	// SkipIfComplete will run Complete() on entry. If Complete() == true,
	// then it'll skip to the next state.
	SkipIfComplete bool

	// Label enables jumping directly to a State with stateMachine.SetState
	Label string
}

// EventRequest is sent to the state machine to request safely jumping between
// states with guards checking that each new state is valid
type EventRequest int

func NewStateMachine(pkgName string) *StateMachine {
	sm := StateMachine{state: 0, pkgName: pkgName}
	sm.states = map[string]int{}
	sm.resetFn = func(*Msg) {}
	sm.logger = log.WithFields(log.Fields{
		"pkg": pkgName,
	})
	return &sm
}

func (sm *StateMachine) SetStates(sss ...[]State) {
	for i, ss := range sss {
		for j, s := range ss {
			sm.Handlers = append(sm.Handlers, s)
			if len(s.Label) > 0 {
				sm.states[s.Label] = i + j
			}
		}
	}
}

func (sm *StateMachine) SetLogger(l *log.Entry) {
	sm.logger = l
}

func (sm *StateMachine) SetDBConn(s *sqlx.DB) {
	sm.db = s
}

func (sm *StateMachine) LoadState(in *Msg) {
	q := `INSERT INTO states
	      (key, userid, value, pkgname) VALUES ($1, $2, $3, $4)
	      ON CONFLICT (userid, key, pkgname) DO UPDATE SET value=$5
	      RETURNING value`
	var val []byte
	err := sm.db.QueryRowx(q, stateKey, in.User.ID, 0, sm.pkgName,
		sm.state).Scan(&val)
	if err != nil && err != sql.ErrNoRows {
		sm.logger.Errorln(err, "fetching value from states")
		sm.state = 0
		return
	}

	// The []byte->string->int conversion is highly inefficient and
	// should be replaced by something faster. There's talk of such
	// []byte->int functions being added to the stdlib.
	//
	// https://github.com/golang/go/issues/2632
	tmp, err := strconv.ParseInt(string(val), 10, 64)
	if err != nil {
		if err.Error() == `strconv.ParseInt: parsing "": invalid syntax` {
			sm.state = 0
			return
		}
		sm.logger.Errorln(err, "parsing state")
	}
	sm.state = int(tmp)
	// Had we already entered a state?
	sm.stateEntered = sm.GetMemory(in, stateEnteredKey).Bool()
	sm.logger.Debugln("set state to", sm.state)
	sm.logger.Debugln("set state entered to", sm.stateEntered)
	return
}

func (sm *StateMachine) State() int {
	return sm.state
}

func (sm *StateMachine) GetDBConn() *sqlx.DB {
	return sm.db
}

func (sm *StateMachine) SetPkgName(n string) {
	sm.pkgName = n
}

func (sm *StateMachine) Next(in *Msg) string {
	h := sm.Handlers[sm.state]
	if sm.state >= len(sm.Handlers) {
		sm.logger.Debugln("state is >= len(handlers)")
		return ""
	}
	if !sm.stateEntered {
		if h.SkipIfComplete {
			done, _ := h.Complete(in)
			if done {
				sm.logger.Debugln("state was complete. moving on")
				return sm.Next(in)
			}
		}
		sm.setEntered(in)
		sm.logger.Debugln("setting state entered")
		return h.OnEntry(in)
	}
	sm.logger.Debugln("state was already entered")
	h.OnInput(in)
	// check completion of current state
	done, str := h.Complete(in)
	if done {
		sm.logger.Debugln("state is done. going to next")
		if sm.state+1 >= len(sm.Handlers) {
			sm.logger.Debugln("finished states, nothing to do")
			return ""
		}
		q := `UPDATE states SET value=$1 WHERE key=$2`
		b := make([]byte, 8) // space for int64
		binary.LittleEndian.PutUint64(b, uint64(sm.state))
		if err, _ := sm.db.Exec(q, b, stateKey); err != nil {
			sm.logger.Errorln(err, "updating state")
		}
		sm.state++
		sm.setEntered(in)
		return sm.Handlers[sm.state].OnEntry(in)
	} else if len(str) > 0 {
		sm.logger.Debugln("incomplete with message")
		return str
	}
	sm.logger.Debugln("reached here")
	return ""
}

func (sm *StateMachine) setEntered(in *Msg) {
	sm.stateEntered = true
	sm.SetMemory(in, stateEnteredKey, true)
	sm.logger.Warnln("set state entered!")
}

func (sm *StateMachine) OnInput(in *Msg) {
	sm.Handlers[sm.state].OnInput(in)
}

func (sm *StateMachine) SetOnReset(reset func(in *Msg)) {
	sm.resetFn = reset
}

func (sm *StateMachine) SetMemory(in *Msg, k string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		sm.logger.Errorln(err, "marhsalling interface to json", v)
		return
	}
	q := `INSERT INTO states (key, value, pkgname, userid)
	      VALUES ($1, $2, $3, $4)
	      ON CONFLICT (userid, pkgname, key) DO UPDATE SET value=$2`
	_, err = sm.db.Exec(q, k, b, sm.pkgName, in.User.ID)
	if err != nil {
		sm.logger.Errorln(err, "setting memory at", k, "to", v)
		return
	}
	// TODO set preference here as well
}

func (sm *StateMachine) GetMemory(in *Msg, k string) Memory {
	q := `SELECT value FROM states WHERE userid=$1 AND pkgname=$2 AND key=$3`
	var buf []byte
	err := sm.db.Get(&buf, q, in.User.ID, sm.pkgName, k)
	if err == sql.ErrNoRows {
		return Memory{Key: k, Val: json.RawMessage{}, logger: sm.logger}
	}
	if err != nil {
		sm.logger.Errorln(err, "getMemory for key", k)
		return Memory{Key: k, Val: json.RawMessage{}, logger: sm.logger}
	}
	return Memory{Key: k, Val: buf, logger: sm.logger}
}

func (sm *StateMachine) HasMemory(in *Msg, k string) bool {
	return len(sm.GetMemory(in, k).Val) > 0
}

func (sm *StateMachine) Reset(in *Msg) {
	sm.state = 0
	sm.stateEntered = false
	sm.SetMemory(in, stateKey, 0)
	sm.SetMemory(in, stateEnteredKey, false)
	sm.resetFn(in)
}

func (sm *StateMachine) SetState(in *Msg, label string) string {
	desiredState := sm.states[label]

	// If we're in a state beyond the desired state, go back. There are NO
	// checks for state, so if you're changing state after its been
	// completed, you'll need to do sanity checks OnEntry.
	if sm.state > desiredState {
		sm.state = desiredState
		sm.stateEntered = false
		return sm.Handlers[desiredState].OnEntry(in)
	}

	// If we're in a state before the desired state, go forward only as far
	// as we're allowed to by the Complete guards
	for s := sm.state; s < desiredState; s++ {
		ok, _ := sm.Handlers[s].Complete(in)
		if !ok {
			sm.state = s
			sm.stateEntered = false
			return sm.Handlers[s].OnEntry(in)
		}
	}

	// No guards were triggered (go to state), or the state == desiredState,
	// so reset the state and run OnEntry again
	sm.state = desiredState
	sm.stateEntered = false
	return sm.Handlers[desiredState].OnEntry(in)
}
