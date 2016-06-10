package dt

// Person is a human being discussed.
type Person struct {
	Name string
	Sex  Sex
}

// Sex represents the sex of the person being discussed.
type Sex int

// Define constants for all possible sexes for the purpose of maintaining
// context in sentences across requests. For example, "Send a bottle of wine to
// Jim and Pam, and tell her that I miss her." Here we can identify the "her"
// most likely refers to Pam.
const (
	SexInvalid Sex = iota
	SexMale
	SexFemale
	SexEither
)
