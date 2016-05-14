package language

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/datatypes"
)

func TestMain(m *testing.M) {
	if err := os.Setenv("ABOT_ENV", "test"); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestExtractCities(t *testing.T) {
	var cities []dt.City
	in := &dt.Msg{}
	db, err := core.ConnectDB()
	if err != nil {
		t.Fatal(err)
	}
	in.Sentence = "I'm in New York"
	in.Tokens = core.TokenizeSentence(in.Sentence)
	in.Stems = core.StemTokens(in.Tokens)
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
	in.Tokens = core.TokenizeSentence(in.Sentence)
	in.Stems = core.StemTokens(in.Tokens)
	cities, err = ExtractCities(db, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(cities) < 2 {
		t.Fatal(fmt.Errorf("expected >2 cities, but got %d\n", len(cities)))
	}
	in = &dt.Msg{}
	in.Sentence = "What's the weather like in San Francisco?"
	in.Tokens = core.TokenizeSentence(in.Sentence)
	in.Stems = core.StemTokens(in.Tokens)
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
