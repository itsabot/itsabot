package language

import (
	"fmt"
	"math/rand"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
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

func FirstMeeting() string {
	n := rand.Intn(3)
	switch n {
	case 0:
		return "Hi, I'm Ava. What's your name?"
	case 1:
		return "Hi, this is Ava. Who is this?"
	case 2:
		return "Hi, my name's Ava. What's your name?"
	}
	log.Error("firstmeeting failed to return a response")
	return ""
}

func NiceMeetingYou() string {
	n := rand.Intn(3)
	switch n {
	case 0:
		return "It's nice to meet you. If we're going to work " +
			"together, can you sign up for me here? "
	case 1:
		return "Nice meeting you. Before we take this further, can " +
			"you sign up for me here? "
	case 2:
		return "Great to meet you! Can you sign up for me here to " +
			"get started? "
	}
	log.Error("nicemeetingyou failed to return a response")
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
	log.Error("suggestedPlace failed to return a response")
	return ""
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
		"burger",
		"eat",
	}
}

// TODO
func USCities() string {
	return "TODO: Not implemented"
}
