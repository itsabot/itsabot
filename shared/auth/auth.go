package auth

import "errors"

// Method allows you as the package developer to control the level of security
// required in an authentication. Select an appropriate security level depending
// upon your risk tolerance for fraud compared against the quality and ease of
// the user experience.
type Method int

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
// recently authenticated. Force will demand a new authentication using the
// current method. Force is useful for large purchases (generally >= $1k), but
// will annoy users and therefore should be avoided on smaller transactions.
func Authenticate(m Method, force bool) error {
	switch m {
	case MethodCVV:

	case MethodZip:
	case MethodWebCache:
	case MethodWebLogin:
	}
	return errors.New("not implemented")
}
