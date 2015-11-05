package datatypes

import "errors"

var (
	ErrInvalidClass        = errors.New("invalid class")
	ErrInvalidOddParameter = errors.New("parameter count must be even")
)

const (
	CommandI int = iota + 1
	ActorI
	ObjectI
	TimeI
	PlaceI
	NoneI
	UnsureI
)

const (
	FlexIdTypeEmail int = iota + 1
	FlexIdTypePhone
)

var String map[int]string = map[int]string{
	CommandI: "Command",
	ActorI:   "Actor",
	ObjectI:  "Object",
	TimeI:    "Time",
	PlaceI:   "Place",
	NoneI:    "None",
}

var Pronouns map[string]int = map[string]int{
	"me":    ActorI,
	"us":    ActorI,
	"you":   ActorI,
	"him":   ActorI,
	"her":   ActorI,
	"them":  ActorI,
	"it":    ObjectI,
	"that":  ObjectI,
	"there": PlaceI,
	"then":  TimeI,
}
