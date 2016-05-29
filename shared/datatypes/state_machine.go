package dt

import "encoding/json"

// StateKey is a reserved key in the state of a plugin that tracks which state
// the plugin is currently in for each user.
const StateKey string = "__state"

// StateKeyEntered keeps track of whether the current state has already been
// "entered", which determines whether the OnEntry function should run or not.
// As mentioned elsewhere, the OnEntry function is only ever run once.
const stateEnteredKey string = "__state_entered"

// StateMachine enables plugin developers to easily build complex state
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
	plugin       *Plugin
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
func NewStateMachine(p *Plugin) *StateMachine {
	sm := StateMachine{
		state:  0,
		plugin: p,
	}
	sm.states = map[string]int{}
	sm.resetFn = func(*Msg) {}
	return &sm
}

// SetStates takes [][]State as an argument. Note that it's a slice of a slice,
// which is used to enable tasks like requesting a user's shipping address,
// which themselves are []Slice, to be included inline when defining the states
// of a stateMachine.
func (sm *StateMachine) SetStates(ssss ...[][]State) {
	for i, sss := range ssss {
		for j, ss := range sss {
			for k, s := range ss {
				sm.Handlers = append(sm.Handlers, s)
				if len(s.Label) > 0 {
					sm.states[s.Label] = i + j + k
				}
			}
		}
	}
}

// LoadState upserts state into the database. If there is an existing state for
// a given user and plugin, the stateMachine will load it. If not, the
// stateMachine will insert a starting state into the database.
func (sm *StateMachine) LoadState(in *Msg) {
	tmp, err := json.Marshal(sm.state)
	if err != nil {
		sm.plugin.Log.Info("failed to marshal state for db.", err)
		return
	}

	// Using upsert to either insert and return a value or on conflict to
	// update and return a value doesn't work, leading to this longer form.
	// Could it be a Postgres bug? This can and should be optimized.
	if in.User.ID > 0 {
		q := `INSERT INTO states
		      (key, userid, value, pluginname) VALUES ($1, $2, $3, $4)`
		_, err = sm.plugin.DB.Exec(q, StateKey, in.User.ID, tmp,
			sm.plugin.Config.Name)
	} else {
		q := `INSERT INTO states
		      (key, flexid, flexidtype, value, pluginname) VALUES ($1, $2, $3, $4, $5)`
		_, err = sm.plugin.DB.Exec(q, StateKey, in.User.FlexID,
			in.User.FlexIDType, tmp, sm.plugin.Config.Name)
	}
	if err != nil {
		if err.Error() != `pq: duplicate key value violates unique constraint "states_userid_pkgname_key_key"` &&
			err.Error() != `pq: duplicate key value violates unique constraint "states_flexid_flexidtype_pluginname_key_key"` {
			sm.plugin.Log.Info("could not insert value into states.", err)
			sm.state = 0
			return
		}
		if in.User.ID > 0 {
			q := `SELECT value FROM states
			      WHERE userid=$1 AND key=$2 AND pluginname=$3`
			err = sm.plugin.DB.Get(&tmp, q, in.User.ID, StateKey,
				sm.plugin.Config.Name)
		} else {
			q := `SELECT value FROM states
			      WHERE flexid=$1 AND flexidtype=$2 AND key=$3 AND pluginname=$4`
			err = sm.plugin.DB.Get(&tmp, q, in.User.FlexID,
				in.User.FlexIDType, StateKey, sm.plugin.Config.Name)
		}
		if err != nil {
			sm.plugin.Log.Info("failed to get value from state.", err)
			return
		}
	}
	var val int
	if err = json.Unmarshal(tmp, &val); err != nil {
		sm.plugin.Log.Info("failed unmarshaling state from db.", err)
		return
	}
	sm.state = val

	// Have we already entered a state?
	sm.stateEntered = sm.plugin.GetMemory(in, stateEnteredKey).Bool()
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

// Next moves a stateMachine from its current state to its next state. Next
// handles a variety of corner cases such as reaching the end of the states,
// ensuring that the current state's Complete() == true, etc. It directly
// returns the next response of the stateMachine, whether that's the Complete()
// failed string or the OnEntry() string.
func (sm *StateMachine) Next(in *Msg) (response string) {
	// This check prevents a panic when no states are being used.
	if len(sm.Handlers) == 0 {
		return
	}

	// This check prevents a panic when a plugin has been modified to remove
	// one or more states.
	if sm.state >= len(sm.Handlers) {
		sm.plugin.Log.Debug("state is >= len(handlers)")
		sm.Reset(in)
	}

	// Ensure the state has not been entered yet
	h := sm.Handlers[sm.state]
	if !sm.stateEntered {
		sm.plugin.Log.Debug("state was not entered")
		done, _ := h.Complete(in)
		if h.SkipIfComplete {
			if done {
				sm.plugin.Log.Debug("state was complete. moving on")
				sm.state++
				sm.plugin.SetMemory(in, StateKey, sm.state)
				return sm.Next(in)
			}
		}
		sm.setEntered(in)
		sm.plugin.Log.Debug("setting state entered")

		// If this is the final state and complete on entry, we'll
		// reset the state machine. This fixes the "forever trapped"
		// loop of being in a plugin's finished state machine.
		resp := h.OnEntry(in)
		if sm.state+1 >= len(sm.Handlers) && done {
			sm.Reset(in)
		}
		return resp
	}

	// State was already entered, so process the input and check for
	// completion
	sm.plugin.Log.Debug("state was already entered")
	h.OnInput(in)
	done, str := h.Complete(in)
	if done {
		sm.plugin.Log.Debug("state is done. going to next")
		sm.state++
		sm.plugin.SetMemory(in, StateKey, sm.state)
		if sm.state >= len(sm.Handlers) {
			sm.plugin.Log.Debug("finished states. resetting")
			sm.Reset(in)
			return sm.Next(in)
		}
		sm.setEntered(in)
		str = sm.Handlers[sm.state].OnEntry(in)
		sm.plugin.Log.Debug("going to next state", sm.state)
		return str
	}

	sm.plugin.Log.Debug("set state to", sm.state)
	sm.plugin.Log.Debug("set state entered to", sm.stateEntered)
	return str
}

// setEntered is used internally to set a state as having been entered both in
// memory and persisted to the database. This ensures that a stateMachine does
// not run a state's OnEntry function twice.
func (sm *StateMachine) setEntered(in *Msg) {
	sm.stateEntered = true
	sm.plugin.SetMemory(in, stateEnteredKey, true)
}

// SetOnReset sets the OnReset function for the stateMachine, which is used to
// clear Abot's memory of temporary things between runs.
func (sm *StateMachine) SetOnReset(reset func(in *Msg)) {
	sm.resetFn = reset
}

// Reset the stateMachine both in memory and in the database. This also runs the
// programmer-defined reset function (SetOnReset) to reset memories to some
// starting state for running the same plugin multiple times.
func (sm *StateMachine) Reset(in *Msg) {
	sm.state = 0
	sm.stateEntered = false
	sm.plugin.SetMemory(in, StateKey, 0)
	sm.plugin.SetMemory(in, stateEnteredKey, false)
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
		sm.plugin.SetMemory(in, StateKey, desiredState)
		sm.plugin.SetMemory(in, stateEnteredKey, false)
		return sm.Handlers[desiredState].OnEntry(in)
	}

	// If we're in a state before the desired state, go forward only as far
	// as we're allowed by the Complete guards.
	for s := sm.state; s < desiredState; s++ {
		ok, _ := sm.Handlers[s].Complete(in)
		if !ok {
			sm.state = s
			sm.stateEntered = false
			sm.plugin.SetMemory(in, StateKey, s)
			sm.plugin.SetMemory(in, stateEnteredKey, false)
			return sm.Handlers[s].OnEntry(in)
		}
	}

	// No guards were triggered (go to state), or the state == desiredState,
	// so reset the state and run OnEntry again unless the plugin is now
	// complete.
	sm.state = desiredState
	sm.stateEntered = false
	sm.plugin.SetMemory(in, StateKey, desiredState)
	sm.plugin.SetMemory(in, stateEnteredKey, false)
	return sm.Handlers[desiredState].OnEntry(in)
}

// ReplayState returns you to the current state's OnEntry function. This is
// only useful when you're iterating over results in a state machine. If you're
// reading this, you should probably be using task.Iterate() instead. If you've
// already considered task.Iterate(), and you've decided to use this underlying
// function instead, you should only call it when Complete returns false, like
// so:
//
// Complete: func(in *dt.Msg) (bool, string) {
//	if p.HasMemory(in, "memkey") {
//		return true, ""
//	}
//	return false, p.SM.ReplayState(in)
// }
//
// That said, the *vast* majority of the time you should NOT be using this
// function. Instead use task.Iterate(), which uses this function safely.
func (sm *StateMachine) ReplayState(in *Msg) string {
	return sm.Handlers[sm.state+1].OnEntry(in)
}
