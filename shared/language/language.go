// Package language does the following four things:
//
// 1. Provides easy-to-use helpers for returning commonly used, randomized text
// such as greetings.
// 2. Normalizes varied user responses like "yup" or "nah" into something to be
// more easily used by packages.
// 3. Consolidates triggers by categories (e.g. automotive brands) if commonly
// used across packages.
// 4. Summarizes text using the custom rule-based algorithm found in
// summarize.go.
package language

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/itsabot/abot/shared/log"
)

var yes map[string]bool = map[string]bool{
	"yes":          true,
	"yea":          true,
	"yeah":         true,
	"yup":          true,
	"yesh":         true,
	"sure":         true,
	"aye":          true,
	"ok":           true,
	"o.k.":         true,
	"that's right": true,
	"thats right":  true,
	"affirmative":  true,
}

var no map[string]bool = map[string]bool{
	"no":          true,
	"nope":        true,
	"nah":         true,
	"not sure":    true,
	"dunno":       true,
	"don't know":  true,
	"do not know": true,
	"negative":    true,
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
	log.Debug("firstmeeting failed to return a response")
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
	log.Debug("nicemeetingyou failed to return a response")
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
	log.Debug("confused failed to return a response")
	return ""
}

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

func Vehicles() []string {
	return []string{
		"car",
		"vehicle",
		"automotive",
		"automobile",
		"motorcycle",
	}
}

func AutomotiveBrands() []string {
	return []string{
		"abarth",
		"acura",
		"alfa",
		"ascari",
		"aston",
		"audi",
		"bentley",
		"bowler",
		"bmw",
		"bugatti",
		"buick",
		"cadillac",
		"caterham",
		"chevrolet",
		"chevy",
		"chrysler",
		"citroen",
		"corvette",
		"datsun",
		"dodge",
		"ferrari",
		"fiat",
		"fisker",
		"ford",
		"gmc",
		"honda",
		"hummer",
		"hyundai",
		"infiniti",
		"isuzu",
		"jaguar",
		"jeep",
		"kia",
		"koenigsegg",
		"ktm",
		"lambo",
		"lamborghini",
		"lancia",
		"rover",
		"lexus",
		"lincoln",
		"lotus",
		"maserati",
		"maybach",
		"mazda",
		"mclaren",
		"merc",
		"mercedes",
		"benz",
		"mg",
		"mini",
		"mitsubishi",
		"nissan",
		"pagani",
		"peugeot",
		"porsche",
		"ram",
		"renault",
		"rolls",
		"rolls-royce",
		"saab",
		"saleen",
		"saturn",
		"scion",
		"seat",
		"skoda",
		"smart",
		"subaru",
		"suzuki",
		"tata",
		"tesla",
		"toyota",
		"tvr",
		"vauxhall",
		"volkswagen",
		"volvo",
	}
}

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

func Repair() []string {
	return []string{
		"repair",
		"repairing",
		"fix",
		"fixing",
	}
}

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

func Yes(s string) bool {
	s = strings.ToLower(s)
	return yes[s]
}

func No(s string) bool {
	s = strings.ToLower(s)
	return no[s]
}

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

var StopWords = []string{
	"a",
	"an",
	"the",
}

var SwearWords = map[string]bool{
	"arse":         true,
	"ass":          true,
	"asshole":      true,
	"bastard":      true,
	"bitch":        true,
	"cunt":         true,
	"damn":         true,
	"fuck":         true,
	"fucker":       true,
	"goddamn":      true,
	"goddamm":      true,
	"goddam":       true,
	"motherfucker": true,
	"shit":         true,
	"whore":        true,
}

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
