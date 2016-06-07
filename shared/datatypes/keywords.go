package dt

import (
	"strings"

	"github.com/dchest/stemmer/porter2"
)

// Keywords maintains sets of Commands and Objects recognized by plugins as well
// as the functions to be performed when such Commands or Objects are found.
type Keywords struct {
	Dict map[string]KeywordFn
}

// KeywordHandler maintains sets of Commands and Objects recognized by plugins as
// well as the functions to be performed when such Commands or Objects are
// found.
type KeywordHandler struct {
	Fn      KeywordFn
	Trigger *StructuredInput
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
		return fn(m)
	}

	// No matching intent was found, so check for both Command and Object.
	eng := porter2.Stemmer
	for _, cmd := range m.StructuredInput.Commands {
		cmd = strings.ToLower(eng.Stem(cmd))
		for _, obj := range m.StructuredInput.Objects {
			obj = strings.ToLower(eng.Stem(obj))
			fn, ok := k.Dict["CO_"+cmd+"_"+obj]
			if !ok {
				continue
			}
			return fn(m)
		}
	}
	return ""
}
