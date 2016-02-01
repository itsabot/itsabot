package dt

import (
	"strconv"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

type Memory struct {
	Key    string
	Val    []byte
	logger *log.Entry
}

func (m Memory) String() string {
	return string(m.Val)
}

func (m Memory) Int64() int64 {
	i, err := strconv.ParseInt(string(m.Val), 10, 64)
	if err != nil && err.Error() != "strconv.ParseInt: parsing \"\"\"\": invalid syntax converting memory to int64" {
		m.logger.Errorln(err, "converting memory to int64", m.Key,
			m.Val)
	}
	return i
}

func (m Memory) Bool() bool {
	b, err := strconv.ParseBool(string(m.Val))
	if err != nil {
		m.logger.Warnln(err, "parsing bool for", m.Key)
	}
	return b
}
