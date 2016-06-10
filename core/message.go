package core

import (
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
)

// NewMsg builds a message struct with Tokens, Stems, and a Structured Input.
func NewMsg(u *dt.User, cmd string) (*dt.Msg, error) {
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
	if err := saveContext(db, m); err != nil {
		return nil, err
	}
	if err := addContext(db, m); err != nil {
		return nil, err
	}
	return m, nil
}
