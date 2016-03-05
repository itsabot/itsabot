package language

import (
	"fmt"
	"testing"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/nlp"
	"github.com/itsabot/abot/shared/plugin"
)

func TestExtractCities(t *testing.T) {
	log.DebugOn(true)
	db, err := plugin.ConnectDB()
	if err != nil {
		t.Error(err)
	}
	var cities []dt.City
	in := &dt.Msg{}
	in.Sentence = "I'm in New York"
	in.Tokens = nlp.TokenizeSentence(in.Sentence)
	in.Stems = nlp.StemTokens(in.Tokens)
	cities, err = ExtractCities(db, in)
	if err != nil {
		t.Error(err)
	}
	if len(cities) == 0 || cities[0].Name != "New York" {
		t.Error(fmt.Errorf("expected New York, but got %s\n", cities[0]))
	}
	in = &dt.Msg{}
	in.Sentence = "I'm in LA or San Francisco next week"
	in.Tokens = nlp.TokenizeSentence(in.Sentence)
	in.Stems = nlp.StemTokens(in.Tokens)
	cities, err = ExtractCities(db, in)
	if err != nil {
		t.Error(err)
	}
	if len(cities) < 2 {
		t.Error(fmt.Errorf("expected >2 cities, but got %d\n", len(cities)))
	}
}
