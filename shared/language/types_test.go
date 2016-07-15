package language

import (
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/datatypes"
)

func TestIsGreeting(t *testing.T) {
	u := &dt.User{ID: 1}
	in, err := core.NewMsg(u, "Hi there")
	if err != nil {
		t.Fatal(err)
	}
	if !IsGreeting(in) {
		t.Fatal("expected greeting")
	}
	in, err = core.NewMsg(u, "Any random sentence.")
	if err != nil {
		t.Fatal(err)
	}
	if IsGreeting(in) {
		t.Fatal("expected not greeting")
	}
}
