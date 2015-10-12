package twilio

import (
	"strconv"
)

type Price float64

func (p *Price) UnmarshalJSON(b []byte) (err error) {
	str := string(b)

	if str == "null" {
		*p = 0
		return nil
	}

	ustr, _ := strconv.Unquote(str)
	f, err := strconv.ParseFloat(ustr, 64)
	if err == nil {
		*p = Price(f)
	}

	return err
}
