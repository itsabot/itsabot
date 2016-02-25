package dt

import (
	log "github.com/Sirupsen/logrus"
	"github.com/dchest/stemmer/porter2"
)

// Vocab maintains sets of Commands and Objects recognized by packages as well
// as the functions to be performed when such Commands or Objects are found.
type Vocab struct {
	Commands map[string]struct{}
	Objects  map[string]struct{}
	dict     map[string]VocabFn
}

// VocabHandler maintains sets of Commands and Objects recognized by packages as well
// as the functions to be performed when such Commands or Objects are found.
//
// TODO use pkg triggers rather than WordTypes, which provides more control over
// when to run specific functions.
type VocabHandler struct {
	Fn       VocabFn
	WordType string
	Words    []string
}

// VocabFn is a function run when the user sends a matched vocab word as defined
// by a package. For an example, see packages/ava_purchase/ava_purchase.go. The
// response returned is a user-presentable string from the VocabFn.
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
		for i := range vh.Words {
			vh.Words[i] = eng.Stem(vh.Words[i])
			v.dict[vh.Words[i]] = vh.Fn
			if vh.WordType == "Command" {
				v.Commands[vh.Words[i]] = struct{}{}
			} else if vh.WordType == "Object" {
				v.Objects[vh.Words[i]] = struct{}{}
			}
		}
	}
	return &v
}

// HandleKeywords is a runs the first matching VocabFn in the sentence. For an
// example, see packages/ava_purchase/ava_purchase.go.
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
