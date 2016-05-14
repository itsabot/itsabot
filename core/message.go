package core

import (
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
)

// newMsg builds a message struct with Tokens, Stems, and a Structured Input.
func newMsg(u *dt.User, cmd string) *dt.Msg {
	tokens := TokenizeSentence(cmd)
	stems := StemTokens(tokens)
	si := ner.classifyTokens(tokens)

	// Get the intents as determined by each plugin
	for pluginID, c := range bClassifiers {
		scores, idx, _ := c.ProbScores(stems)
		log.Debug("intent score", pluginIntents[pluginID][idx],
			scores[idx])
		if scores[idx] > 0.7 {
			si.Intents = append(si.Intents,
				string(pluginIntents[pluginID][idx]))
		}
	}

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
