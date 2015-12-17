package dt

import (
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"strconv"
	"strings"
)

type Uint64_Slice []uint64

func (s *Uint64_Slice) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("scan source was not []bytes"))
	}
	str := string(asBytes)
	str = quoteEscapeRegex.ReplaceAllString(str, `$1""`)
	str = strings.Replace(str, `\\`, `\`, -1)
	csvReader := csv.NewReader(strings.NewReader(str))
	slice, err := csvReader.Read()
	if err != nil && err.Error() != "EOF" {
		return err
	}
	*s = Uint64_Slice(slice)
	return nil
}

func (s Uint64_Slice) Value() (driver.Value, error) {
	var vals []string
	for i, elem := range s {
		vals[i] = strconv.FormatUint(elem, 10)
	}
	return "{" + strings.Join(vals, ",") + "}", nil
}
