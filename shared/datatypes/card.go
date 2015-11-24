package dt

type Card struct {
	ID             int
	AddressID      int
	Last4          string
	CardholderName string
	ExpMonth       int
	ExpYear        int
	Brand          string
	StripeID       string
}
