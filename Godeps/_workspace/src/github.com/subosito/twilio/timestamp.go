package twilio

import (
	"strconv"
	"time"
)

type Timestamp struct {
	time.Time
}

func (t *Timestamp) UnmarshalJSON(b []byte) (err error) {
	str := string(b)

	if str == "null" {
		(*t).Time = time.Time{}
		return nil
	}

	var ustr string

	ustr, err = strconv.Unquote(str)
	if err != nil {
		ustr = str
	}

	m, err := time.Parse(time.RFC1123Z, ustr)

	if err == nil {
		(*t).Time = m
	} else {
		(*t).Time = time.Time{}
	}

	return nil
}

func (t *Timestamp) IsZero() bool {
	return t.Time.IsZero()
}

func (t Timestamp) Equal(m Timestamp) bool {
	return t.Time.Equal(m.Time)
}
