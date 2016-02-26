package dt

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"strconv"

	"github.com/itsabot/abot/shared/log"
	"github.com/jmoiron/sqlx"
)

// stateKey is a reserved key in the state of a package that tracks which state
// the package is currently in for each user.
const stateKey string = "__state"

// stateKeyEntered keeps track of whether the current state has already been
// "entered", which determines whether the OnEntry function should run or not.
// As mentioned elsewhere, the OnEntry function is only ever run once.
const stateEnteredKey string = "__state_entered"

// stateMachine enables package developers to easily build complex state
// machines given the constraints and use-cases of an A.I. bot. It primarily
// holds a slice of function Handlers, which is all possible states for a given
// stateMachine. The unexported variables are useful internally in keeping track
// of state automatically for developers and make an easy API like
// stateMachine.Next() possible.
type StateMachine struct {
	Handlers     []State
	state        int
	stateEntered bool
	states       map[string]int
	keys         []string
	pkgName      string
	logger       *log.Logger
	db           *sqlx.DB
	resetFn      func(*Msg)
}

// State is a collection of pre-defined functions that are run when a user
// reaches the appropriate state within a stateMachine.
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

	// Label enables jumping directly to a State with stateMachine.SetState.
	// Think of it as enabling a safe goto statement. This is especially
	// useful when combined with a KeywordHandler, enabling a user to jump
	// straight to something like a "checkout" state. The state machine
	// checks before jumping that it has all required information before
	// jumping ensuring Complete() == true at all skipped states, so the
	// developer can be sure, for example, that the user has selected some
	// products and picked a shipping address before arriving at the
	// checkout step. In the case where one of the jumped Complete()
	// functions returns false, the state machine will stop at that state,
	// i.e. as close to the desired state as possible.
	Label string
}

// EventRequest is sent to the state machine to request safely jumping between
// states (directly to a specific Label) with guards checking that each new
// state is valid.
type EventRequest int

// NewStateMachine initializes a stateMachine to its starting state.
func NewStateMachine(pkgName string) *StateMachine {
	sm := StateMachine{state: 0, pkgName: pkgName}
	sm.states = map[string]int{}
	sm.resetFn = func(*Msg) {}
	sm.logger = log.New(pkgName)
	return &sm
}

// SetStates takes [][]State as an argument. Note that it's a slice of a slice,
// which is used to enable tasks like requesting a user's shipping address,
// which themselves are []Slice, to be included inline when defining the states
// of a stateMachine. See packages/ava_purchase/ava_purchase.go as an example.
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

// SetLogger enables the logger with any package-defined settings to be used
// internally by the stateMachine. This ensures consistency in the logs of a
// package.
func (sm *StateMachine) SetLogger(l *log.Logger) {
	sm.logger = l
}

// SetDBConn gives a stateMachine itsabot.org/abot/shared access to a package's database
// connection. This is required even if no states require database access, since
// the stateMachine's current state (among other things) are peristed to the
// database between user requests.
func (sm *StateMachine) SetDBConn(s *sqlx.DB) {
	sm.db = s
}

// SetOnReset sets the OnReset function for the stateMachine, which should be
// called from a package's Run() function. See
// packages/ava_purchase/ava_purchase.go for an example.
func (sm *StateMachine) SetOnReset(reset func(in *Msg)) {
	sm.resetFn = reset
}

// LoadState upserts state into the database. If there is an existing state for
// a given user and package, the stateMachine will load it. If not, the
// stateMachine will insert a starting state into the database.
func (sm *StateMachine) LoadState(in *Msg) {
	q := `INSERT INTO states
	      (key, userid, value, pkgname) VALUES ($1, $2, $3, $4)
	      ON CONFLICT (userid, key, pkgname) DO UPDATE SET value=$5
	      RETURNING value`
	var val []byte
	err := sm.db.QueryRowx(q, stateKey, in.User.ID, 0, sm.pkgName,
		sm.state).Scan(&val)
	if err != nil && err != sql.ErrNoRows {
		sm.logger.Debug("could not fetch value from states", err)
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
		sm.logger.Debug("could not parse state", err)
	}
	sm.state = int(tmp)
	// Have we already entered a state?
	sm.stateEntered = sm.GetMemory(in, stateEnteredKey).Bool()
	sm.logger.Debug("set state to", sm.state)
	sm.logger.Debug("set state entered to", sm.stateEntered)
	return
}

// State returns the current state of a stateMachine. state is an unexported
// field to protect programmers from directly editing it. While reading state
// can be done through this function, changing state should happen only through
// the provided stateMachine API (stateMachine.Next(), stateMachine.SetState()),
// which allows for safely moving between states.
func (sm *StateMachine) State() int {
	return sm.state
}

// GetDBConn allows for accessing a stateMachine's provided database connection.
func (sm *StateMachine) GetDBConn() *sqlx.DB {
	return sm.db
}

// Next moves a stateMachine from its current state to its next state. Next
// handles a variety of corner cases such as reaching the end of the states,
// ensuring that the current state's Complete() == true, etc. It directly
// returns the next response of the stateMachine, whether that's the Complete()
// failed string or the OnEntry() string.
func (sm *StateMachine) Next(in *Msg) (response string) {
	h := sm.Handlers[sm.state]
	if sm.state >= len(sm.Handlers) {
		sm.logger.Debug("state is >= len(handlers)")
		return ""
	}
	if !sm.stateEntered {
		if h.SkipIfComplete {
			done, _ := h.Complete(in)
			if done {
				sm.logger.Debug("state was complete. moving on")
				return sm.Next(in)
			}
		}
		sm.setEntered(in)
		sm.logger.Debug("setting state entered")
		return h.OnEntry(in)
	}
	sm.logger.Debug("state was already entered")
	h.OnInput(in)
	// Check completion of current state
	done, str := h.Complete(in)
	if done {
		sm.logger.Debug("state is done. going to next")
		if sm.state+1 >= len(sm.Handlers) {
			sm.logger.Debug("finished states, nothing to do")
			return ""
		}
		q := `UPDATE states SET value=$1 WHERE key=$2`
		b := make([]byte, 8) // space for int64
		binary.LittleEndian.PutUint64(b, uint64(sm.state))
		if err, _ := sm.db.Exec(q, b, stateKey); err != nil {
			sm.logger.Debug("could not update state", err)
		}
		sm.state++
		sm.setEntered(in)
		return sm.Handlers[sm.state].OnEntry(in)
	} else if len(str) > 0 {
		sm.logger.Debug("incomplete with message")
		return str
	}
	sm.logger.Debug("reached here")
	return ""
}

// setEntered is used internally to set a state as having been entered both in
// memory and persisted to the database. This ensures that a stateMachine does
// not run a state's OnEntry function twice.
func (sm *StateMachine) setEntered(in *Msg) {
	sm.stateEntered = true
	sm.SetMemory(in, stateEnteredKey, true)
}

// OnInput runs the stateMachine's current OnInput function. Most of the time
// this is not used directly, since Next() will automatically run this function
// when appropriate. It's an exported function to provide users more control
// over their
func (sm *StateMachine) OnInput(in *Msg) {
	sm.Handlers[sm.state].OnInput(in)
}

// SetMemory saves to some key to some value in Ava's memory, which can be
// accessed by any state or package. Memories are stored in a key-value format,
// and any marshalable/unmarshalable datatype can be stored and retrieved.
// Note that Ava's memory is global, peristed across packages. This enables
// packages that subscribe to an agreed-upon memory API to communicate between
// themselves. Thus, if it's absolutely necessary that no some other packages
// modify or access a memory, use a long key unlikely to collide with any other
// package's.
func (sm *StateMachine) SetMemory(in *Msg, k string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		sm.logger.Debug("could not marshal memory interface to json at",
			k, ":", err)
		return
	}
	q := `INSERT INTO states (key, value, pkgname, userid)
	      VALUES ($1, $2, $3, $4)
	      ON CONFLICT (userid, pkgname, key) DO UPDATE SET value=$2`
	_, err = sm.db.Exec(q, k, b, sm.pkgName, in.User.ID)
	if err != nil {
		sm.logger.Debug("could not set memory at", k, "to", v, ":", err)
		return
	}
}

// GetMemory retrieves a memory for a given key. Accessing that Memory's value
// is described in itsabot.org/abot/shared/datatypes/memory.go.
func (sm *StateMachine) GetMemory(in *Msg, k string) Memory {
	q := `SELECT value FROM states WHERE userid=$1 AND pkgname=$2 AND key=$3`
	var buf []byte
	err := sm.db.Get(&buf, q, in.User.ID, sm.pkgName, k)
	if err == sql.ErrNoRows {
		return Memory{Key: k, Val: json.RawMessage{}, logger: sm.logger}
	}
	if err != nil {
		sm.logger.Debug("could not get memory for key", k, ":", err)
		return Memory{Key: k, Val: json.RawMessage{}, logger: sm.logger}
	}
	return Memory{Key: k, Val: buf, logger: sm.logger}
}

// HasMemory is a helper function to simply a common use-case, determing if some
// key/value has been set in Ava, i.e. if the memory exists.
func (sm *StateMachine) HasMemory(in *Msg, k string) bool {
	return len(sm.GetMemory(in, k).Val) > 0
}

// Reset the stateMachine both in memory and in the database. This also runs the
// programmer-defined reset function (SetOnReset) to reset memories to some
// starting state for running the same package multiple times. This is usually
// called from a package's Run() function. See
// packages/ava_purchase/ava_purchase.go for an example.
func (sm *StateMachine) Reset(in *Msg) {
	sm.state = 0
	sm.stateEntered = false
	sm.SetMemory(in, stateKey, 0)
	sm.SetMemory(in, stateEnteredKey, false)
	sm.resetFn(in)
}

// SetState jumps from one state to another by its label. It will safely jump
// forward but NO safety checks are performed on backward jumps. It's therefore
// up to the developer to ensure that data is still OK when jumping backward.
// Any forward jump will check the Complete() function of each state and get as
// close as it can to the desired state as long as each Complete() == true at
// each state.
func (sm *StateMachine) SetState(in *Msg, label string) string {
	desiredState := sm.states[label]

	// If we're in a state beyond the desired state, go back. There are NO
	// checks for state when going backward, so if you're changing state
	// after its been completed, you'll need to do sanity checks OnEntry.
	if sm.state > desiredState {
		sm.state = desiredState
		sm.stateEntered = false
		return sm.Handlers[desiredState].OnEntry(in)
	}

	// If we're in a state before the desired state, go forward only as far
	// as we're allowed to by the Complete guards.
	for s := sm.state; s < desiredState; s++ {
		ok, _ := sm.Handlers[s].Complete(in)
		if !ok {
			sm.state = s
			sm.stateEntered = false
			return sm.Handlers[s].OnEntry(in)
		}
	}

	// No guards were triggered (go to state), or the state == desiredState,
	// so reset the state and run OnEntry again.
	sm.state = desiredState
	sm.stateEntered = false
	return sm.Handlers[desiredState].OnEntry(in)
}
