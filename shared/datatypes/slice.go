package dt

import (
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"regexp"
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

// StringSlice replaces []string, adding custom sql support for arrays in lieu
// of pq.
type StringSlice []string

// QuoteEscapeRegex replaces escaped quotes except if it is preceded by a
// literal backslash, e.g. "\\" should translate to a quoted element whose value
// is \
var QuoteEscapeRegex = regexp.MustCompile(`([^\\]([\\]{2})*)\\"`)

// Scan converts to a slice of strings. See:
// http://www.postgresql.org/docs/9.1/static/arrays.html#ARRAYS-IO
func (s *StringSlice) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return errors.New("scan source was not []bytes")
	}
	str := string(asBytes)
	str = QuoteEscapeRegex.ReplaceAllString(str, `$1""`)
	str = strings.Replace(str, `\\`, `\`, -1)
	str = str[1 : len(str)-1]
	csvReader := csv.NewReader(strings.NewReader(str))
	slice, err := csvReader.Read()
	if err != nil && err.Error() != "EOF" {
		return err
	}
	*s = StringSlice(slice)
	return nil
}

// Value converts to a slice of strings. See:
// http://www.postgresql.org/docs/9.1/static/arrays.html#ARRAYS-IO
func (s StringSlice) Value() (driver.Value, error) {
	// string escapes.
	// \ => \\\
	// " => \"
	for i, elem := range s {
		s[i] = `"` + strings.Replace(strings.Replace(elem, `\`, `\\\`, -1), `"`, `\"`, -1) + `"`
	}
	return "{" + strings.Join(s, ",") + "}", nil
}

// Last safely returns the last item in a StringSlice, which is most often the
// target of a pronoun, e.g. (In "Where is that?", "that" will most often refer
// to the last Object named in the previous sentence.
func (s StringSlice) Last() string {
	if len(s) == 0 {
		return ""
	}
	return s[len(s)-1]
}

// String converts a StringSlice into a string with each word separated by
// spaces.
func (s StringSlice) String() string {
	if len(s) == 0 {
		return ""
	}
	var ss string
	for _, w := range s {
		ss += " " + w
	}
	return ss[1:]
}

// StringSlice converts a StringSlice into a []string.
func (s StringSlice) StringSlice() []string {
	ss := []string{}
	for _, tmp := range s {
		if len(tmp) <= 2 {
			continue
		}
		ss = append(ss, tmp)
	}
	return ss
}

// Map converts a StringSlice into a map to check quickly if words exist within
// it.
func (s StringSlice) Map() map[string]struct{} {
	m := map[string]struct{}{}
	for _, w := range s {
		m[w] = struct{}{}
	}
	return m
}
