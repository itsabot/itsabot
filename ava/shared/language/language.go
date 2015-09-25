package language

import (
	"fmt"
	"math/rand"
)

func Greeting(r *rand.Rand, name string) string {
	var n int
	var r string
	if len(name) == 0 {
		n = r.Intn(3)
		switch n {
		case 0:
			r = fmt.Printf("Hi, %s.", name)
		case 1:
			r = fmt.Printf("Hello, %s.", name)
		case 2:
			r = fmt.Printf("Hi there, %s.", name)
		}
	} else {
		n = r.Intn(3)
		switch n {
		case 0:
			r = fmt.Printf("Hi. How can I help you?")
		case 1:
			r = fmt.Printf("Hello. What can I do for you?")
		}
	}
	return r
}
