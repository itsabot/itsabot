package dt

import "github.com/itsabot/abot/shared/nlp"

// Keywords maintains sets of Commands and Objects recognized by plugins as well
// as the functions to be performed when such Commands or Objects are found.
type Keywords struct {
	Commands map[string]struct{}
	Objects  map[string]struct{}
	Intents  map[string]struct{}
	Dict     map[string]*KeywordFn
}

// KeywordHandler maintains sets of Commands and Objects recognized by plugins as
// well as the functions to be performed when such Commands or Objects are
// found.
type KeywordHandler struct {
	Fn      KeywordFn
	Trigger *nlp.StructuredInput
}

// KeywordFn is a function run when the user sends a matched keyword as
// defined by a plugin. The response returned is a user-presentable string from
// the KeywordFn.
type KeywordFn func(in *Msg) (response string)

// handle runs the first matching KeywordFn in the sentence.
func (k *Keywords) handle(m *Msg) string {
	if k == nil {
		return ""
	}
	for _, intent := range m.StructuredInput.Intents {
		fn, ok := k.Dict["I_"+intent]
		if !ok {
			continue
		}

		// If we find an intent function, use that.
		return (*fn)(m)
	}

	// No matching intent was found, so check for both Command and Object.
	var fns []*KeywordFn
	for _, cmd := range m.StructuredInput.Commands {
		fn, ok := k.Dict["C_"+cmd]
		if !ok {
			continue
		}
		fns = append(fns, fn)
	}
	if len(fns) == 0 {
		return ""
	}
	idx := -1
	for _, obj := range m.StructuredInput.Objects {
		fn2, ok := k.Dict["O_"+obj]
		if !ok {
			continue
		}
		for i, fn := range fns {
			if fn2 == fn {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return ""
	}
	return (*fns[idx])(m)
}
