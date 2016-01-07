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
	if err != nil {
		m.logger.Errorln(err, "converting memory to int64", m.Key,
			m.Val)
	}
	return i
}
