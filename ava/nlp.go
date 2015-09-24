package main

import (
	"strings"

	"github.com/jbrukh/bayesian"
)

const (
	Command bayesian.Class = "Command"
	Actor   bayesian.Class = "Actor"
	Object  bayesian.Class = "Object"
	Time    bayesian.Class = "Time"
	None    bayesian.Class = "None"
)

const (
	CommandI int = iota
	ActorI
	ObjectI
	TimeI
	NoneI
)

type StructuredInput struct {
	Sentence string
	Command  []string
	Actors   []string
	Objects  []string
	Times    []string
}

type wordclass struct {
	Word  string
	Class int
}

func buildStructuredInput(nl string) StructuredInput {
	var si StructuredInput

	/*
		sentences := strings.Split(nl, ".")
		for _, sent := range sentences {
			si = si.add(parseSentence(sent))
		}
	*/
	return si
}

func (si *StructuredInput) String() string {
	s := "\n"
	s += "Command: " + strings.Join(si.Command, ", ") + "\n"
	s += "Actors: " + strings.Join(si.Actors, ", ") + "\n"
	s += "Objects: " + strings.Join(si.Objects, ", ") + "\n"
	s += "Times: " + strings.Join(si.Times, ", ") + "\n"
	return s
}

// Add pairs of words with their classes to a structured input. Params should
// follow the ("Order", "Command"), ("noon", "Time") form.
func (si *StructuredInput) Add(wc []wordclass) error {
	if len(wc) == 0 {
		return ErrInvalidOddParameter
	}
	for _, w := range wc {
		switch w.Class {
		case CommandI:
			si.Command = append(si.Command, w.Word)
		case ActorI:
			si.Actors = append(si.Actors, w.Word)
		case ObjectI:
			si.Objects = append(si.Objects, w.Word)
		case TimeI:
			si.Times = append(si.Times, w.Word)
		case NoneI:
			// Do nothing
		default:
			return ErrInvalidClass
		}
	}
	return nil
}
