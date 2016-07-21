package core

import (
	"os"
	"reflect"
	"testing"

	"github.com/itsabot/abot/core/log"
)

func TestSpellCheckTokens(t *testing.T) {
	if err := os.Setenv("ABOT_ENV", "test"); err != nil {
		t.Fatal(err)
	}
	if err := trainSpellCheck(); err != nil {
		t.Fatal(err)
	}
	tokens := []string{"let", "'", "is", "try", "to", "break", "this"}
	checked := spellCheckTokens(tokens)
	expected := []string{"let", "'", "is", "try", "to", "break", "this"}
	if !reflect.DeepEqual(checked, expected) {
		log.Info("expected", expected, "received", checked)
		t.Fail()
	}
	tokens = []string{"this", "is", "an", "incorect", "spellng"}
	checked = spellCheckTokens(tokens)
	expected = []string{"this", "is", "an", "incorrect", "spelling"}
	if !reflect.DeepEqual(checked, expected) {
		log.Info("expected", expected, "received", checked)
		t.Fail()
	}
	tokens = []string{"does", "ths", "wrk", "with", "comon", "wrds"}
	checked = spellCheckTokens(tokens)
	expected = []string{"does", "this", "work", "with", "common", "words"}
	if !reflect.DeepEqual(checked, expected) {
		log.Info("expected", expected, "received", checked)
		t.Fail()
	}
	tokens = []string{"bill", "sarah", "jane", "oft", "obtain", "procure", "alcoholic", "beverages"}
	expected = tokens
	checked = spellCheckTokens(tokens)
	if !reflect.DeepEqual(checked, expected) {
		log.Info("expected", expected, "received", checked)
		t.Fail()
	}
}
