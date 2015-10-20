package language

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
)

var yes map[string]bool = map[string]bool{
	"yes":          true,
	"yea":          true,
	"yeah":         true,
	"yup":          true,
	"sure":         true,
	"that's right": true,
	"thats right":  true,
	"think so":     true,
}

var no map[string]bool = map[string]bool{
	"no":          true,
	"nope":        true,
	"nah":         true,
	"not sure":    true,
	"dunno":       true,
	"don't know":  true,
	"do not know": true,
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
	log.Println("greeting failed to return a response")
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
	log.Println("firstmeeting failed to return a response")
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
	log.Println("nicemeetingyou failed to return a response")
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
	log.Println("confused failed to return a response")
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
	log.Println("positive failed to return a response")
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
	log.Println("welcome failed to return a response")
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
	log.Println("suggestedPlace failed to return a response")
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

func Broken() []string {
	return []string{
		"broke",
		"broken",
		"help",
		"start",
	}
}

func QuestionLocation(loc string) string {
	if len(loc) == 0 {
		n := rand.Intn(6)
		switch n {
		case 0:
			return "Where are you?"
		case 1:
			return "Sure. Where are you?"
		case 2:
			return "Happy to help. Where are you?"
		case 3:
			return "Sure thing. Where are you?"
		case 4:
			return "Can do. Where are you?"
		case 5:
			return "I can help with that. Where are you?"
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
