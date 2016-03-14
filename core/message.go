package core

import (
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/nlp"
)

// NewMsg builds a message struct with Tokens, Stems, and a Structured Input.
func NewMsg(u *dt.User, cmd string) *dt.Msg {

	tokens := nlp.TokenizeSentence(cmd)
	stems := nlp.StemTokens(tokens)
	si := NER().ClassifyTokens(tokens)
	m := &dt.Msg{
		User:            u,
		Sentence:        cmd,
		Tokens:          tokens,
		Stems:           stems,
		StructuredInput: si,
	}
	/*
		m, err = addContext(db, m)
		if err != nil {
			log.Debug(err)
		}
	*/
	return m
}
