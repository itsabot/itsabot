package language

import (
	"github.com/itsabot/abot/shared/datatypes"
)

func IsGreeting(in *dt.Msg) bool {
	for _, stem := range in.Stems {
		switch stem {
		case "hi", "hello", "greete":
			return true
		}
	}
	// TODO handle cases involving bigrams like "what's up?" or "what's
	// going (on)?" Extend dt.Msg to do that efficiently with msg.Bigrams.
	return false
}
