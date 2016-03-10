package core

import (
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/nlp"
	"github.com/jmoiron/sqlx"
)

// NewMsg builds a message struct with Tokens, Stems, and a Structured Input.
func NewMsg(db *sqlx.DB, classifier Classifier, u *dt.User,
	cmd string) *dt.Msg {

	tokens := nlp.TokenizeSentence(cmd)
	stems := nlp.StemTokens(tokens)
	si := classifier.ClassifyTokens(tokens)
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
