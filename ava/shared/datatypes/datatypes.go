package datatypes

import (
	"errors"
	"strings"
)

var (
	ErrInvalidClass        = errors.New("invalid class")
	ErrInvalidOddParameter = errors.New("parameter count must be even")
)

const (
	CommandI int = iota
	ActorI
	ObjectI
	TimeI
	PlaceI
	NoneI
)

var String map[int]string = map[int]string{
	CommandI: "Command",
	ActorI:   "Actor",
	ObjectI:  "Object",
	TimeI:    "Time",
	PlaceI:   "Place",
	NoneI:    "None",
}

type StructuredInput struct {
	Id       string
	Sentence string
	Command  []string
	Actors   []string
	Objects  []string
	Times    []string
	Places   []string
}

func (si *StructuredInput) String() string {
	s := "\n"
	if len(si.Command) > 0 {
		s += "Command: " + strings.Join(si.Command, ", ") + "\n"
	}
	if len(si.Actors) > 0 {
		s += "Actors: " + strings.Join(si.Actors, ", ") + "\n"
	}
	if len(si.Objects) > 0 {
		s += "Objects: " + strings.Join(si.Objects, ", ") + "\n"
	}
	if len(si.Times) > 0 {
		s += "Times: " + strings.Join(si.Times, ", ") + "\n"
	}
	if len(si.Places) > 0 {
		s += "Places: " + strings.Join(si.Places, ", ") + "\n"
	}
	return s[:len(s)-1] + "\n"
}

type WordClass struct {
	Word  string
	Class int
}

// Add pairs of words with their classes to a structured input. Params should
// follow the ("Order", "Command"), ("noon", "Time") form.
func (si *StructuredInput) Add(wc []WordClass) error {
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
		case PlaceI:
			si.Places = append(si.Places, w.Word)
		case NoneI:
			// Do nothing
		default:
			return ErrInvalidClass
		}
	}
	return nil
}
