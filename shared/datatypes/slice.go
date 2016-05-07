package dt

import (
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"
)

// Uint64Slice extends []uint64 with support for arrays in pq.
type Uint64Slice []uint64

// Scan converts to a slice of uint64.
func (u *Uint64Slice) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return errors.New("scan source was not []bytes")
	}
	str := string(asBytes)
	str = str[1 : len(str)-1]
	csvReader := csv.NewReader(strings.NewReader(str))
	slice, err := csvReader.Read()
	if err != nil && err.Error() != "EOF" {
		return err
	}
	var s []uint64
	for _, sl := range slice {
		tmp, err := strconv.ParseUint(sl, 10, 64)
		if err != nil {
			return err
		}
		s = append(s, tmp)
	}
	*u = Uint64Slice(s)
	return nil
}

// Value converts to a slice of uint64.
func (u Uint64Slice) Value() (driver.Value, error) {
	var ss []string
	for i := 0; i < len(u); i++ {
		tmp := strconv.FormatUint(u[i], 10)
		ss = append(ss, tmp)
	}
	return "{" + strings.Join(ss, ",") + "}", nil
}
