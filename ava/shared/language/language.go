package language

import (
	"fmt"
	"math/rand"
)

func Greeting(r *rand.Rand, name string) string {
	var n int
	var resp string
	if len(name) == 0 {
		n = r.Intn(3)
		switch n {
		case 0:
			resp = fmt.Sprintf("Hi, %s.", name)
		case 1:
			resp = fmt.Sprintf("Hello, %s.", name)
		case 2:
			resp = fmt.Sprintf("Hi there, %s.", name)
		}
	} else {
		n = r.Intn(3)
		switch n {
		case 0:
			resp = "Hi. How can I help you?"
		case 1:
			resp = "Hello. What can I do for you?"
		}
	}
	return resp
}

// TODO: Extend
func Foods() []string {
	return []string{
		"food",
		"restaurant",
		"restaurants",
		"pizza",
		"chinese",
		"japanese",
		"korean",
		"asian",
		"italian",
		"ramen",
		"to eat",
	}
}

// TODO
func USCities() string {
	return "TODO: Not implemented"
}
