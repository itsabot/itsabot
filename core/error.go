package core

import (
	"errors"
	"strings"
)

// JSONError builds a simple JSON message from an error type in the format of
// { "Msg": err.Error() }. This ensures that any client expecting a JSON error
// message (e.g. Abot's web front-end) receives one.
func JSONError(err error) error {
	tmp := strings.Replace(err.Error(), `"`, "'", -1)
	return errors.New(`{"Msg":"` + tmp + `"}`)
}
