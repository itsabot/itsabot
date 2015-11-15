package auth

import (
	"regexp"
	"time"

	"github.com/avabot/ava/shared/datatypes"
)

// Method allows you as the package developer to control the level of security
// required in an authentication. Select an appropriate security level depending
// upon your risk tolerance for fraud compared against the quality and ease of
// the user experience.
type Method int

var regexNum = regexp.MustCompile(`\d+`)

const (
	// MethodCVV will require the CVV (3-4 digit security code) for a credit
	// card on file. If the user has no credit cards on file, the user will
	// be asked for one.
	MethodCVV = Method{iota + 1}

	// MethodZip requires the zip code associated with a credit card on
	// file. Just like MethodCVV, the user will be asked for a credit card
	// if not on file. This method is considered slightly more secure than
	// CVV, since having the physical credit card (and therefore the CVV) is
	// not enough to make a purchase.
	MethodZip

	// MethodWebCache allows a user to authenticate by clicking a link. If
	// their browser cookies have them already logged into Ava, they will be
	// authenticated. If they are not currently logged into Ava, they will
	// be asked to login. Once logged in, they will be authenticated.
	MethodWebCache

	// MethodWebLogin requires the user login to Ava on the web interface
	// using their username and password. This is the most secure option,
	// as it ensures no one has stolen the device or session token of a
	// user.
	MethodWebLogin
)

// Authenticate ensures you're speaking to the correct user. Select the LOWEST
// level of authentication you'll allow based on a tolerance for fraud weighed
// against the convenience of the user experience. Methods are organized in
// least-secure to most-secure order. Therefore, MethodCVV will allow any
// authentication method, whereas MethodZip will only allow MethodZip and above.
// Ava will IMPROVE the quality of the authentication automatically whenever
// possible, selecting the highest authentication method for which the user has
// recently authenticated. Note that you'll never have to call Authenticate in a
// Purchase flow. In order to drive a customer purchase, call Purchase directly,
// which will also authenticate the user.
func Authenticate(m Method, u *datatypes.User) (bool, error) {
	// check last authentication date and method
	if err := u.GetLastAuthentication(); err != nil {
		return false, err
	}
	yesterday := time.Now().Add(time.Duration(time.Hour) * -24)
	if u.LastAuthenticated != nil && u.LastAuthenticated.After(yesterday) {
	}
	authenticated, method, err := getLastAuthentication()
	if err != nil {
		return false, err
	}
	if authenticated && int(method) >= int(m) {
		return true, nil
	}
	switch m {
	case MethodCVV:
		cards, err := getCards()
		if err != nil {
			return err
		}
		t := "Please confirm a card's security code (CVC)"
		// send user confirmation text
		// handle response
	case MethodZip:
		cards, err := getCards()
		if err != nil {
			return err
		}
		t := "Please confirm your billing zip code"
	case MethodWebCache:
		t := "Please prove you're logged in: https://www.avabot.com/?/profile"
	case MethodWebLogin:
		if err := deleteUserSession(); err != nil {
			return err
		}
		t := "Please log in to prove it's you: https://www.avabot.com/?/login"
	}
	return false, nil
}

// Purchase will authenticate the user and then charge a card.
func Purchase(m Method, price uint64, card *datatypes.Card) error {
	if err := Authenticate(m); err != nil {
		return err
	}

	return nil
}
