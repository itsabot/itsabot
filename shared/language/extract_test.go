package language

import (
	"errors"
	"fmt"
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/nlp"
)

func TestExtractCities(t *testing.T) {
	log.SetDebug(true)
	var cities []dt.City
	in := &dt.Msg{}
	db, err := core.ConnectDB()
	if err != nil {
		t.Fatal(err)
	}
	in.Sentence = "I'm in New York"
	in.Tokens = nlp.TokenizeSentence(in.Sentence)
	in.Stems = nlp.StemTokens(in.Tokens)
	cities, err = ExtractCities(db, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cities) == 0 {
		t.Fatal(errors.New("expected New York, extracted none"))
	}
	if cities[0].Name != "New York" {
		t.Fatal(fmt.Errorf("expected New York, extracted %s", cities[0].Name))
	}
	in = &dt.Msg{}
	in.Sentence = "I'm in LA or San Francisco next week"
	in.Tokens = nlp.TokenizeSentence(in.Sentence)
	in.Stems = nlp.StemTokens(in.Tokens)
	cities, err = ExtractCities(db, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cities) < 2 {
		t.Fatal(fmt.Errorf("expected >2 cities, but got %d\n", len(cities)))
	}
	in = &dt.Msg{}
	in.Sentence = "What's the weather like in San Francisco?"
	in.Tokens = nlp.TokenizeSentence(in.Sentence)
	in.Stems = nlp.StemTokens(in.Tokens)
	cities, err = ExtractCities(db, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cities) == 0 {
		t.Fatal(fmt.Errorf("expected San Francisco, extracted none"))
	}
	if cities[0].Name != "San Francisco" {
		t.Fatal(fmt.Errorf("expected San Francisco, extracted %s", cities[0].Name))
	}
}
