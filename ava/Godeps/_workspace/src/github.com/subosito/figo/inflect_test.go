package figo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPluralizeCount(t *testing.T) {
	table := map[int][]string{
		0: []string{"person", "people"},
		1: []string{"person", "person"},
		2: []string{"person", "people"},
	}

	for k, v := range table {
		assert.Equal(t, PluralizeCount(k, v[0]), v[1])
	}
}

func TestPluralizeWithCount(t *testing.T) {
	table := map[int][]string{
		0: []string{"person", "0 people"},
		1: []string{"person", "1 person"},
		2: []string{"person", "2 people"},
	}

	for k, v := range table {
		assert.Equal(t, PluralizeWithCount(k, v[0]), v[1])
	}
}
