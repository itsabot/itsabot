package core

import (
	"strconv"
	"strings"
	"testing"
)

type test struct {
	Test  string
	Class string
	Exp   []string
}

func TestClassifyTokens(t *testing.T) {
	ner, err := buildClassifier()
	if err != nil {
		t.Fatal(err)
	}
	tests := []test{
		{
			Test:  "Where is Jim?",
			Class: "person",
			Exp:   []string{"Jim"},
		},
		{
			Test:  "Who's pam?",
			Class: "person",
			Exp:   []string{"pam"},
		},
		{
			Test:  "i'll be there at noon",
			Class: "time",
			Exp:   []string{"12"},
		},
		{
			Test:  "order pizza",
			Class: "command",
			Exp:   []string{"order"},
		},
		{
			Test:  "find the post office",
			Class: "command",
			Exp:   []string{"find", "post"},
		},
		{
			Test:  "order pizza",
			Class: "object",
			Exp:   []string{"order", "pizza"},
		},
		{
			Test:  "find the post office",
			Class: "object",
			Exp:   []string{"find", "post", "office"},
		},
	}
	for _, test := range tests {
		tokens := TokenizeSentence(test.Test)
		si := ner.classifyTokens(tokens)
		switch test.Class {
		case "command":
			if len(si.Commands) == 0 {
				er(t, test.Exp, "", test.Test)
			}
			if len(si.Commands) != len(test.Exp) {
				er(t, test.Exp, strings.Join(si.Commands, ", "), test.Test)
			}
			for i, exp := range test.Exp {
				if si.Commands[i] != exp {
					er(t, test.Exp, si.Commands[0], test.Test)
				}
			}
		case "object":
			if len(si.Objects) == 0 {
				er(t, test.Exp, "", test.Test)
			}
			if len(si.Objects) != len(test.Exp) {
				er(t, test.Exp, strings.Join(si.Objects, ", "), test.Test)
			}
			for i, exp := range test.Exp {
				if si.Objects[i] != exp {
					er(t, test.Exp, si.Objects[i], test.Test)
				}
			}
		case "time":
			if len(si.Times) == 0 {
				er(t, test.Exp, "", test.Test)
			}
			for i, exp := range test.Exp {
				hour := strconv.Itoa(si.Times[i].Hour())
				if hour != exp {
					er(t, test.Exp, hour, test.Test)
				}
			}
		case "person":
			if len(si.People) == 0 {
				er(t, test.Exp, "", test.Test)
			}
			if len(si.People) != len(test.Exp) {
				var names []string
				for _, person := range si.People {
					names = append(names, person.Name)
				}
				er(t, test.Exp, strings.Join(names, ", "), test.Test)
			}
			for i, exp := range test.Exp {
				if si.People[i].Name != exp {
					er(t, test.Exp, si.People[i].Name, test.Test)
				}
			}
		}
	}
}

func er(t *testing.T, expected []string, received, test string) {
	t.Fatalf("expected %q, received %q for %q\n", expected, received, test)
}
