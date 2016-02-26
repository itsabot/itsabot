package figo

import (
	"bitbucket.org/pkg/inflect"
	"fmt"
)

// PluralizeCount returns the plural or singular form depends on c.
// If c is 1 it will returns singular form, otherwise plural form is returned.
func PluralizeCount(c int, s string) string {
	return pluralize(c, s, false)
}

// PluralizeWithCount returns the plural or singular form depends on c and returns c as part of the returned string.
// If c is 1 it will returns singular form, otherwise plural form is returned.
func PluralizeWithCount(c int, s string) string {
	return pluralize(c, s, true)
}

func pluralize(c int, s string, format bool) string {
	if c == 1 {
		if format {
			return fmt.Sprintf("1 %s", inflect.Singularize(s))
		}

		return fmt.Sprintf("%s", inflect.Singularize(s))
	}

	if format {
		return fmt.Sprintf("%d %s", c, inflect.Pluralize(s))
	}

	return fmt.Sprintf("%s", inflect.Pluralize(s))
}
