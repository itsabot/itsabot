package dt

import (
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/nlp"
)

// Keywords maintains sets of Commands and Objects recognized by plugins as well
// as the functions to be performed when such Commands or Objects are found.
type Keywords struct {
	Commands map[string]struct{}
	Objects  map[string]struct{}
	Intents  map[string]struct{}
	Dict     map[string]KeywordFn
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
	var resp string
	for _, w := range m.Stems {
		if k.Dict[w] == nil {
			continue
		}
		log.Debug("found fn in stems", w, k.Dict[w])
		resp = (k.Dict[w])(m)
		break
	}
	return resp
}
