package dt

import (
	"github.com/dchest/stemmer/porter2"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/nlp"
)

// Vocab maintains sets of Commands and Objects recognized by plugins as well
// as the functions to be performed when such Commands or Objects are found.
type Vocab struct {
	Commands map[string]struct{}
	Objects  map[string]struct{}
	dict     map[string]VocabFn
}

// VocabHandler maintains sets of Commands and Objects recognized by plugins as
// well as the functions to be performed when such Commands or Objects are
// found.
type VocabHandler struct {
	Fn      VocabFn
	Trigger *nlp.StructuredInput
}

// VocabFn is a function run when the user sends a matched vocab word as
// defined by a plugin. For an example, see
// github.com/itsabot/plugin_purchase/plugin_purchase.go. The response returned
// is a user-presentable string from the VocabFn.
type VocabFn func(m *Msg, mod int) (response string)

// NewVocab returns Vocab with all Commands and Objects stemmed using the
// Porter2 Snowball algorithm.
func NewVocab(vhs ...VocabHandler) *Vocab {
	v := Vocab{
		Commands: map[string]struct{}{},
		Objects:  map[string]struct{}{},
		dict:     map[string]VocabFn{},
	}
	eng := porter2.Stemmer
	for _, vh := range vhs {
		for _, cmd := range vh.Trigger.Commands {
			v.dict[cmd] = vh.Fn
			cmd = eng.Stem(cmd)
			v.Commands[cmd] = struct{}{}
		}
		for _, obj := range vh.Trigger.Objects {
			v.dict[obj] = vh.Fn
			obj = eng.Stem(obj)
			v.Objects[obj] = struct{}{}
		}
	}
	return &v
}

// HandleKeywords is a runs the first matching VocabFn in the sentence. For an
// example, see github.com/itsabot/plugin_purchase/plugin_purchase.go.
func (v *Vocab) HandleKeywords(m *Msg) string {
	var resp string
	mod := 1
	for _, w := range m.Stems {
		if v.dict[w] == nil {
			continue
		}
		log.Debug("found fn in stems", w, v.dict[w])
		resp = (v.dict[w])(m, mod)
		break
	}
	return resp
}
