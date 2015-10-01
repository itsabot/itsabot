package language

import (
	"fmt"
	"math/rand"

	log "github.com/Sirupsen/logrus"
)

func Greeting(r *rand.Rand, name string) string {
	var n int
	if len(name) == 0 {
		n = r.Intn(3)
		switch n {
		case 0:
			return fmt.Sprintf("Hi, %s.", name)
		case 1:
			return fmt.Sprintf("Hello, %s.", name)
		case 2:
			return fmt.Sprintf("Hi there, %s.", name)
		}
	} else {
		n = r.Intn(3)
		switch n {
		case 0:
			return "Hi. How can I help you?"
		case 1:
			return "Hello. What can I do for you?"
		}
	}
	log.Error("greeting failed to return a response")
	return ""
}

func Confused() string {
	n := rand.Intn(4)
	switch n {
	case 0:
		return "I'm not sure I understand you."
	case 1:
		return "I'm sorry, I don't understand that."
	case 2:
		return "Uh, what are you telling me to do?"
	case 3:
		return "What should I do?"
	}
	log.Error("confused failed to return a response")
	return ""
}

func SuggestedPlace(s string) string {
	n := rand.Intn(4)
	switch n {
	case 0:
		return "How does this place look? " + s
	case 1:
		return "How about " + s + "?"
	case 2:
		return "Have you been here before? " + s
	case 3:
		return "You could try this: " + s
	}
}

// TODO: Extend
func Foods() []string {
	return []string{
		"food",
		"restaurant",
		"restaurants",
		"chinese",
		"japanese",
		"korean",
		"asian",
		"italian",
		"ramen",
		"pizza",
		"to eat",
	}
}

// TODO
func USCities() string {
	return "TODO: Not implemented"
}
