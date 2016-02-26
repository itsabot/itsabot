package templates

import (
	"fmt"
	"os"
)

// ForgotPasswordEmail takes a user's name and a secret token stored in the
// database and returns an HTML-format email
func ForgotPasswordEmail(name, secret string) string {
	h := `<html><body>`
	h += fmt.Sprintf("<p>Hi %s:</p>", name)
	h += "<p>Please click the following link to reset your password. This link will expire in 30 minutes.</p>"
	h += fmt.Sprintf("<p>%s</p>", os.Getenv("ABOT_URL")+"?/reset_password?s="+secret)
	h += "<p>If you received this email in error, please ignore it.</p>"
	h += "<p>Have a great day!</p>"
	h += "<p>-Ava</p>"
	h += "</body></html>"
	return h
}
