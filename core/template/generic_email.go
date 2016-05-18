package template

import "fmt"

// GenericEmail wraps Abot's response with typical email boilerplate, like a
// greeting (e.g. "Hi Jeff:") and a signature.
func GenericEmail(name, paragraphs []string) string {
	h := "<html><body>"
	if len(name) > 0 {
		h += fmt.Sprintf("<p>Hi %s:</p>", name)
	}
	for _, p := range paragraphs {
		h += fmt.Sprintf("<p>%s</p>", p)
	}
	h += "<p>Best,</p>"
	h += "<p>-Abot</p>"
	h += "</body></html>"
	return h
}
