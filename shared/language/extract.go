package language

import (
	"regexp"
	"strings"
)

var regexCurrency = regexp.MustCompile(`\d+\.?\d*`)

func ExtractCurrency(s string) (string, bool) {
	found := regexCurrency.FindString(s)
	found = strings.Replace(found, ".", "", 1)
	return found, len(found) > 0
}

func ExtractYesNo(s string) (bool, bool) {
	ss := strings.Fields(s)
	for _, w := range ss {
		if yes[w] {
			return true, true
		}
		if no[w] {
			return false, true
		}
	}
	return false, false
}
