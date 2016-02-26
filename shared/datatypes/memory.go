package dt

import (
	"strconv"

	"github.com/itsabot/abot/shared/log"
)

// Memory holds a generic "memory" of Ava's usually set by a package, such as
// the current state of a package, selected products, results of a search,
// current offset in those search results, etc. Since the value is returned as a
// a []byte (and stored in the database in the same way), it can represent any
// datatype, and it's up to the package developer to recall which memories
// correspond to which datatypes.
type Memory struct {
	Key    string
	Val    []byte
	logger *log.Logger
}

// String is a helper method making it easier to perform a common use-case,
// converting a memory's []byte Val into a string.
func (m Memory) String() string {
	return string(m.Val)
}

// Int64 is a helper method making it easier to perform a common use-case,
// converting a memory's []byte Val into an int64 and protecting against a
// common error.
func (m Memory) Int64() int64 {
	i, err := strconv.ParseInt(string(m.Val), 10, 64)
	if err != nil && err.Error() != "strconv.ParseInt: parsing \"\"\"\": invalid syntax converting memory to int64" {
		m.logger.Debugf("could not convert memory to int64 %s (%s, %s)",
			err, m.Key, string(m.Val))
	}
	return i
}

// Bool is a helper method making it easier to perform a common use-case,
// converting a memory's []byte Val into bool and protecting against a common
// error.
func (m Memory) Bool() bool {
	b, err := strconv.ParseBool(string(m.Val))
	if err != nil {
		m.logger.Debugf("could not convert memory to bool %s (%s, %s)",
			err, m.Key, string(m.Val))
	}
	return b
}
