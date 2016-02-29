package core

import (
	"errors"
	"strings"
)

// JSONError builds a simple JSON message from an error type in the format of
// { "Msg": err.Error() }
func JSONError(err error) error {
	tmp := strings.Replace(err.Error(), `"`, "'", -1)
	return errors.New(`{"Msg":"` + tmp + `"}`)
}
