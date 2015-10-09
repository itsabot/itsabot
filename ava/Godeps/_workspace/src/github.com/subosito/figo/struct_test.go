package figo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type StructTest struct {
	Int         int
	Uint        uint
	Float32     float32
	Float64     float64
	Bool        bool
	String      string
	SliceString []string
	SliceInt    []int
}

func TestStructToMapString(t *testing.T) {
	st := StructTest{
		Int:         -134020434,
		Uint:        4039455774,
		Float32:     0.06563702,
		Float64:     0.6868230728671094,
		Bool:        true,
		String:      "hello",
		SliceString: []string{"foo", "bar"},
		SliceInt:    []int{1, 2, 3},
	}

	w := map[string][]string{
		"Int":         []string{"-134020434"},
		"Uint":        []string{"4039455774"},
		"Float32":     []string{"0.0656"},
		"Float64":     []string{"0.6868"},
		"Bool":        []string{"true"},
		"String":      []string{"hello"},
		"SliceString": []string{"foo", "bar"},
		"SliceInt":    []string{"1", "2", "3"},
	}

	assert.Equal(t, w, StructToMapString(&st))
}
