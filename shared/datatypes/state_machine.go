package dt

type StateMachine struct {
	State    int
	Handlers []State
	Reset    *func()
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
	Complete func() bool

	// Memory will search through preferences about the user. If a past
	// preference is found, it'll skip to the OnInput response, with that
	// preference as the input.
	Memory prefs.Key
}

func NewStateMachine(ss ...State) *StateMachine {
	sm := StateMachine{State: 0}
	sm.Handlers = ss
	sm.Reset = func() {}
	return &sm
}

func (sm StateMachine) Next() string {
	if sm.State+1 >= len(sm.Handlers) {
		sm.Reset()
		return sm.Handlers[sm.State].OnEntry()
	}
	if sm.Handlers[sm.State].Complete() {
		sm.State++
		return sm.Handlers[sm.State].OnEntry()
	}
	return ""
}

func (sm StateMachine) OnInput(in *Input) {
	sm.Handlers[sm.State].OnInput(in)
}

func (sm StateMachine) OnReset(reset func()) {
	sm.Reset = reset
}

func (sm StateMachine) Reset() {
	sm.State = 0
	sm.Reset()
}
