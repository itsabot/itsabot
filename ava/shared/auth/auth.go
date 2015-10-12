package auth

import (
	"errors"

	log "github.com/Sirupsen/logrus"
)

type Method int

// TODO finish docs for these
const (
	// MethodTapLink ensures that the user sent the message. In SMS, the
	// user will receive a link to click. By clicking it, you can be more
	// sure that the user requested the action, rather than someone with a
	// spoofed number or email.
	MethodTapLink Method = iota + 1

	// MethodCVV will require the CVV (3-4 digit security code) for a credit
	// card on file. If the user has no credit cards on file, the user will
	// be asked for one.
	MethodCVV

	// MethodZip requires the zip code associated with a credit card on
	// file. Just like MethodCVV, the user will be asked for a credit card
	// if not on file. This method is considered slightly more secure than
	// CVV, since having the credit card (and therefore the CVV) is not
	// enough.
	MethodZip

	// TODO MethodWeb: require a traditional username and password
)

// Authenticate ensures you're speaking to the correct user. Select the LOWEST
// level of authentication you'll allow based on a tolerance for fraud weighed
// against the convenience of the user experience. Methods are organized in
// least-secure to most-secure order. Therefore, MethodTapLink will allow any
// authentication method, whereas MethodZip will only allow MethodZip and above.
// Ava will IMPROVE the quality of the authentication automatically whenever
// possible, selecting the highest authentication method for which the user has
// previously authenticated. TODO implement
func Authenticate(m Method) (Method, error) {
	log.Error("Authenticate not implemented")
	return Method(0), errors.New("not implemented")
}
