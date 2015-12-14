package dt

import (
	"errors"

	"github.com/dchest/stemmer/porter2"
)

type Vocab struct {
	Commands map[string]bool
	Objects  map[string]bool
	dict     map[string]*VocabFn
}

type VocabHandler struct {
	Fn       VocabFn
	WordType string
	Words    []string
}

type VocabFn func(ctx *Ctx, mod int) (string, error)

var ErrNoFn = errors.New("no function")

func NewVocab(vhs ...VocabHandler) *Vocab {
	v := Vocab{
		Commands: map[string]bool{},
		Objects:  map[string]bool{},
		dict:     map[string]*VocabFn{},
	}
	eng := porter2.Stemmer
	for _, vh := range vhs {
		for i := range vh.Words {
			vh.Words[i] = eng.Stem(vh.Words[i])
			v.dict[vh.Words[i]] = &vh.Fn
			if vh.WordType == "Command" {
				v.Commands[vh.Words[i]] = true
			} else if vh.WordType == "Object" {
				v.Objects[vh.Words[i]] = true
			}
		}
		b := bigrams(vh.Words)
		for _, w := range b {
			v.dict[w] = &vh.Fn
		}
	}
	return &v
}

func (v *Vocab) HandleKeywords(ctx *Ctx, resp *Resp, stems []string) error {
	var err error
	b := bigrams(stems)
	mod := 1
	for _, w := range b {
		if v.dict[w] == nil {
			continue
		}
		resp.Sentence, err = (*v.dict[w])(ctx, mod)
		break
	}
	if len(resp.Sentence) == 0 {
		for _, w := range stems {
			if v.dict[w] == nil {
				continue
			}
			resp.Sentence, err = (*v.dict[w])(ctx, mod)
			break
		}
	}
	return err
}

func bigrams(words []string) []string {
	bigrams := []string{}
	for i := 0; i < len(words)-1; i++ {
		bigrams = append(bigrams, words[i]+" "+words[i+1])
	}
	return bigrams
}
