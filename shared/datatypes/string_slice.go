package datatypes

import (
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"regexp"
	"strings"
)

// StringSlice replaces []string, adding custom sql support for arrays in lieu
// of pq.
type StringSlice []string

// quoteEscapeRegex replaces escaped quotes except if it is preceded by a
// literal backslash, e.g. "\\" should translate to a quoted element whose value
// is \
var quoteEscapeRegex = regexp.MustCompile(`([^\\]([\\]{2})*)\\"`)

// Scan convert to a slice of strings
// http://www.postgresql.org/docs/9.1/static/arrays.html#ARRAYS-IO
func (s *StringSlice) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("scan source was not []bytes"))
	}
	str := string(asBytes)
	str = quoteEscapeRegex.ReplaceAllString(str, `$1""`)
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

func (s StringSlice) StringSlice() []string {
	ss := []string{}
	for _, tmp := range s {
		ss = append(ss, tmp)
	}
	return ss
}
