package dt

import (
	"errors"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/dchest/stemmer/porter2"
)

type Vocab struct {
	Commands map[string]bool
	Objects  map[string]bool
	dict     map[string]VocabFn
}

type VocabHandler struct {
	Fn       VocabFn
	WordType string
	Words    []string
}

type VocabFn func(m *Msg, mod int) string

var ErrNoFn = errors.New("no function")

func NewVocab(vhs ...VocabHandler) *Vocab {
	v := Vocab{
		Commands: map[string]bool{},
		Objects:  map[string]bool{},
		dict:     map[string]VocabFn{},
	}
	eng := porter2.Stemmer
	for _, vh := range vhs {
		for i := range vh.Words {
			vh.Words[i] = eng.Stem(vh.Words[i])
			v.dict[vh.Words[i]] = vh.Fn
			if vh.WordType == "Command" {
				v.Commands[vh.Words[i]] = true
			} else if vh.WordType == "Object" {
				v.Objects[vh.Words[i]] = true
			}
		}
	}
	return &v
}

func (v *Vocab) HandleKeywords(m *Msg) string {
	var resp string
	mod := 1
	for _, w := range m.Stems {
		if v.dict[w] == nil {
			continue
		}
		log.Println("found fn in stems", w, v.dict[w])
		resp = (v.dict[w])(m, mod)
		break
	}
	return resp
}
