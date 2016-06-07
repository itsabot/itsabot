// Package language makes it easier to build plugins and natural-sounding
// responses. This package does the following four things:
//
//	1. Provides easy-to-use helpers for returning commonly used, randomized
//	text such as greetings.
//	2. Normalizes varied user responses like "yup" or "nah" into something
//	to be more easily used by plugins.
//	3. Consolidates triggers by categories (e.g. automotive brands) if
//	commonly used across plugins.
//	4. Summarizes text using the custom rule-based algorithm found in
//	summarize.go.
package language

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/itsabot/abot/core/log"
)

var yes = map[string]struct{}{
	"yes":          struct{}{},
	"yea":          struct{}{},
	"yah":          struct{}{},
	"yeah":         struct{}{},
	"yup":          struct{}{},
	"yesh":         struct{}{},
	"sure":         struct{}{},
	"aye":          struct{}{},
	"ok":           struct{}{},
	"o.k.":         struct{}{},
	"k":            struct{}{},
	"kk":           struct{}{},
	"that's right": struct{}{},
	"thats right":  struct{}{},
	"affirmative":  struct{}{},
	"perfect":      struct{}{},
}

var no = map[string]struct{}{
	"no":          struct{}{},
	"nope":        struct{}{},
	"nah":         struct{}{},
	"not sure":    struct{}{},
	"dunno":       struct{}{},
	"don't know":  struct{}{},
	"do not know": struct{}{},
	"negative":    struct{}{},
}

// Join concatenates triggers together, like Recommend() and Broken(), ensuring
// no duplicates exist
func Join(ss ...[]string) []string {
	used := map[string]bool{}
	var s []string
	for _, tmp := range ss {
		for _, w := range tmp {
			if !used[w] {
				s = append(s, w)
			}
			used[w] = true
		}
	}
	return s
}

// Greeting returns a randomized greeting.
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
	log.Debug("greeting failed to return a response")
	return ""
}

// Positive returns a randomized positive response to a user message.
func Positive() string {
	n := rand.Intn(3)
	switch n {
	case 0:
		return "Great!"
	case 1:
		return "I'm glad to hear that!"
	case 2:
		return "Great to hear!"
	}
	log.Debug("positive failed to return a response")
	return ""
}

// Welcome returns a randomized "you're welcome" response to a user message.
func Welcome() string {
	n := rand.Intn(5)
	switch n {
	case 0:
		return "You're welcome!"
	case 1:
		return "Sure thing!"
	case 2:
		return "I'm happy to help!"
	case 3:
		return "My pleasure."
	case 4:
		return "Sure."
	}
	log.Debug("welcome failed to return a response")
	return ""
}

// SuggestedPlace returns a randomized place suggestion useful for recommending
// restaurants, businesses, etc.
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
	log.Debug("suggestedPlace failed to return a response")
	return ""
}

// SuggestedProduct returns natural language, randomized text for a product
// suggestion.
func SuggestedProduct(s string, num uint) string {
	var n int
	var val, flair string
	if num > 0 {
		n = rand.Intn(3)
		switch n {
		case 0, 1:
			flair = ", then"
		case 2: // do nothing
		}
	}
	n = rand.Intn(6)
	switch n {
	case 0:
		val = "I found just the thing"
	case 1:
		val = "This is the one for you"
	case 2:
		val = "You'll love this"
	case 3:
		val = "This is a real treat"
	case 4:
		val = "This will amaze you"
	case 5:
		val = "I found just the thing for you"
	}
	return val + flair + ". " + s
}

// Foods returns a list of foods useful in a plugin's object triggers.
func Foods() []string {
	return []string{
		"almonds",
		"apple",
		"artichokes",
		"asparagus",
		"avocado",
		"bacon",
		"banana",
		"beef",
		"blueberries",
		"burger",
		"butter",
		"can",
		"candy",
		"cashew",
		"cheddar",
		"cheese",
		"chicken",
		"chocolate",
		"cream",
		"donuts",
		"eggs",
		"fish",
		"food",
		"grapes",
		"halibut",
		"ham",
		"hamburger",
		"ice",
		"lamb",
		"mango",
		"meatball",
		"mozzarella",
		"mushroom",
		"noodles",
		"oyster",
		"pancakes",
		"pasta",
		"peanut",
		"pineapple",
		"pizza",
		"pop",
		"popcorn",
		"pork",
		"potato",
		"prawns",
		"ramen",
		"rice",
		"salmon",
		"shrimp",
		"soda",
		"soup",
		"steak",
		"strawberries",
		"sushi",
		"sweetcorn",
		"tomato",
		"tuna",
		"turkey",
		"waffle",
		"watermelon",
	}
}

// Desserts returns a []string of dessert types useful in a
// plugin's object triggers
func Desserts() []string {
	return []string{
		"bar",
		"brownie",
		"cake",
		"cheesecake",
		"chocolate",
		"cobbler",
		"cookie",
		"crepe",
		"cupcake",
		"frosting",
		"macaroon",
		"muffin",
		"pie",
		"shortbread",
		"shortcake",
		"square",
		"tart",
	}
}

// Restaurants returns a []string of restaurant types useful
// in a plugin's object triggers
func Restaurants() []string {
	return []string{
		"american",
		"asian",
		"bakery",
		"bar",
		"barbecue",
		"bbq",
		"bistro",
		"bouchon",
		"brasserie",
		"breakfast",
		"buffet",
		"burger",
		"cafe",
		"cafeteria",
		"cakery",
		"carvery",
		"chinese",
		"coffeehouse",
		"concession",
		"cosplay",
		"cuisine",
		"diner",
		"dining",
		"dinner",
		"drive-in",
		"eat",
		"fast",
		"food",
		"french",
		"greasy",
		"haute",
		"health",
		"italian",
		"izakaya",
		"japanese",
		"korean",
		"lunch",
		"meadery",
		"milk",
		"ouzeri",
		"pancake",
		"parlor",
		"pizza",
		"pub",
		"ramen",
		"restaurant",
		"roadhouse",
		"sandwich",
		"seafood",
		"snack",
		"soda",
		"soup",
		"steakhouse",
		"stop",
		"take-out",
		"tavern",
		"thai",
		"theme",
		"trattoria",
		"truck",
	}
}

// Transportation returns a list of vehicle types useful in a plugin's
// object triggers.
func Transportation() []string {
	return []string{
		"SUV",
		"automobile",
		"automotive",
		"bike",
		"bus",
		"cab",
		"car",
		"motorcycle",
		"scooter",
		"taxi",
		"truck",
		"van",
		"vehicle",
	}
}

// Recommend returns a slice of words related to recommending a product, which
// is useful in a plugin's command trigger.
func Recommend() []string {
	return []string{
		"find",
		"show",
		"where",
		"recommend",
		"recommended",
		"recommendation",
		"recommendations",
	}
}

// Repair returns a slice of words related to repairing something, which is
// useful in a plugin's command trigger.
func Repair() []string {
	return []string{
		"repair",
		"repairing",
		"fix",
		"fixing",
	}
}

// Broken returns a slice of words related to something breaking, which is
// useful in a plugin's command trigger.
func Broken() []string {
	return []string{
		"broke",
		"broken",
		"breaking",
		"help",
		"start",
		"stop",
		"stopped",
		"stopping",
	}
}

// Purchase returns a slice of words related to purchasing something, which is
// useful in a plugin's command trigger.
func Purchase() []string {
	return []string{
		"find",
		"send",
		"get",
		"buy",
		"order",
		"purchase",
		"recommend",
		"recommendation",
		"recommendations",
		"want",
		"finish",
		"complete",
		"cancel",
	}
}

// QuestionLocation returns a randomized question asking a user where they are.
func QuestionLocation(loc string) string {
	if len(loc) == 0 {
		n := rand.Intn(10)
		switch n {
		case 0:
			return "Where are you?"
		case 1:
			return "Where are you now?"
		case 2:
			return "Sure thing. Where are you?"
		case 3:
			return "Sure thing. Where are you now?"
		case 4:
			return "Happy to help. Where are you?"
		case 5:
			return "Happy to help. Where are you now?"
		case 6:
			return "I can help with that. Where are you?"
		case 7:
			return "I can help with that. Where are you now?"
		case 8:
			return "I can help solve this. Where are you?"
		case 9:
			return "I can help solve this. Where are you now?"
		}
	}
	return fmt.Sprintf("Are you still near %s?", loc)
}

// Yes determines if a specific word is a positive "Yes" response. For example,
// "yeah" returns true.
func Yes(s string) bool {
	s = strings.ToLower(s)
	_, ok := yes[s]
	return ok
}

// No determines if a specific word is a "No" response. For example, "nah"
// returns true.
func No(s string) bool {
	s = strings.ToLower(s)
	_, ok := no[s]
	return ok
}

// SliceToString converts a slice of strings into a natural-language list with
// appropriately placed commas and a custom and/or separator.
func SliceToString(ss []string, andor string) string {
	l := len(ss)
	if l == 0 {
		return ""
	}
	if l == 1 {
		return ss[0]
	}
	if l == 2 {
		if andor == "." {
			tmp := strings.Title(ss[1][:1]) + ss[1][1:]
			return fmt.Sprintf("%s%s %s", ss[0], andor, tmp)
		}
		return fmt.Sprintf("%s %s %s", ss[0], andor, ss[1])
	}
	var ret string
	// TODO handle andor == "."
	for i, s := range ss {
		if i == l-2 {
			ret += fmt.Sprintf("%s %s", s, andor)
		} else if i == l-1 {
			ret += " " + s
		} else {
			ret += s + ", "
		}
	}
	return ret
}

// StopWords are articles that can be ignored by Abot.
var StopWords = []string{
	"a",
	"an",
	"the",
}

// Prepositions contains the most commonly used prepositions.
var Prepositions = map[string]struct{}{
	"aboard":      struct{}{},
	"about":       struct{}{},
	"above":       struct{}{},
	"across":      struct{}{},
	"after":       struct{}{},
	"against":     struct{}{},
	"along":       struct{}{},
	"amid":        struct{}{},
	"among":       struct{}{},
	"anti":        struct{}{},
	"around":      struct{}{},
	"as":          struct{}{},
	"at":          struct{}{},
	"before":      struct{}{},
	"behind":      struct{}{},
	"below":       struct{}{},
	"beneath":     struct{}{},
	"beside":      struct{}{},
	"besides":     struct{}{},
	"between":     struct{}{},
	"beyond":      struct{}{},
	"but":         struct{}{},
	"by":          struct{}{},
	"concerning":  struct{}{},
	"considering": struct{}{},
	"despite":     struct{}{},
	"down":        struct{}{},
	"during":      struct{}{},
	"except":      struct{}{},
	"excepting":   struct{}{},
	"excluding":   struct{}{},
	"following":   struct{}{},
	"for":         struct{}{},
	"from":        struct{}{},
	"in":          struct{}{},
	"inside":      struct{}{},
	"into":        struct{}{},
	"like":        struct{}{},
	"minus":       struct{}{},
	"near":        struct{}{},
	"of":          struct{}{},
	"off":         struct{}{},
	"on":          struct{}{},
	"onto":        struct{}{},
	"opposite":    struct{}{},
	"outside":     struct{}{},
	"over":        struct{}{},
	"past":        struct{}{},
	"per":         struct{}{},
	"plus":        struct{}{},
	"regarding":   struct{}{},
	"round":       struct{}{},
	"save":        struct{}{},
	"since":       struct{}{},
	"than":        struct{}{},
	"through":     struct{}{},
	"to":          struct{}{},
	"toward":      struct{}{},
	"towards":     struct{}{},
	"under":       struct{}{},
	"underneath":  struct{}{},
	"unlike":      struct{}{},
	"until":       struct{}{},
	"up":          struct{}{},
	"upon":        struct{}{},
	"versus":      struct{}{},
	"via":         struct{}{},
	"with":        struct{}{},
	"within":      struct{}{},
	"without":     struct{}{},
}

// RemoveStopWords finds and removes stopwords from a slice of strings.
func RemoveStopWords(ss []string) []string {
	var removal []int
	for i, s := range ss {
		if Contains(StopWords, s) {
			removal = append(removal, i)
		}
	}
	for _, i := range removal {
		ss = append(ss[:i], ss[i+1:]...)
	}
	return ss
}

// NiceMeetingYou is used to greet the user and request signup during an
// onboarding process.
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
	log.Debug("nicemeetingyou failed to return a response")
	return ""
}
