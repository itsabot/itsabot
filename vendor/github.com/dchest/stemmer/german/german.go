// Copyright 2012 The Stemmer Package Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package german implements German stemmer, as described in
// http://snowball.tartarus.org/algorithms/german/stemmer.html
package german

import (
	"github.com/dchest/stemmer"
	"strings"
)

// Stemmer is a global, shared instance of German stemmer.
var Stemmer stemmer.Stemmer = germanStemmer(true)

type germanStemmer bool

func suffixPos(s, suf []rune) int {
	if len(s) < len(suf) {
		return -1
	}
	j := len(s) - 1
	for i := len(suf) - 1; i >= 0; i-- {
		if suf[i] != s[j] {
			return -1
		}
		j--
	}
	return len(s) - len(suf)
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'y', 'ä', 'ö', 'ü':
		return true
	}
	return false
}

func calcR(s []rune) int {
	for i := 0; i < len(s)-1; i++ {
		if isVowel(s[i]) && !isVowel(s[i+1]) {
			return i + 2
		}
	}
	return len(s)
}

func adjustR1(s []rune, r1 int) int {
	if r1 >= 3 {
		return r1
	}
	if len(s) < 4 {
		return len(s)
	}
	return 3
}

func getR1R2(s []rune) (r1, r2 int) {
	r1 = calcR(s)
	r2 = r1 + calcR(s[r1:])
	r1 = adjustR1(s, r1)
	return
}

func hasValidSEnding(s []rune) bool {
	last := s[len(s)-1]
	switch last {
	case 'b', 'd', 'f', 'g', 'h', 'k', 'l', 'm', 'n', 'r', 't':
		return true
	}
	return false
}

func hasValidStEnding(s []rune) bool {
	last := s[len(s)-1]
	switch last {
	case 'b', 'd', 'f', 'g', 'h', 'k', 'l', 'm', 'n', 't':
		return true
	}
	return false
}

// Stem returns a stemmed string word.
func (stm germanStemmer) Stem(word string) string {
	word = strings.ToLower(word)
	word = strings.Replace(word, "ß", "ss", -1)

	s := []rune(word)

	for i, max := 1, len(s)-1; i < max; i++ { // interested only in runes between vowels, so we run from 1 to len-1
		if s[i] == 'y' && isVowel(s[i-1]) && isVowel(s[i+1]) {
			s[i] = 'Y'
		}
		if s[i] == 'u' && isVowel(s[i-1]) && isVowel(s[i+1]) {
			s[i] = 'U'
		}
	}

	r1, r2 := getR1R2(s)

	// step 1 group a
	i := suffixPos(s, []rune("ern"))
	if i == -1 {
		i = suffixPos(s, []rune("em"))
	}
	if i == -1 {
		i = suffixPos(s, []rune("er"))
	}
	if i >= r1 {
		s = s[:i]
		goto step2
	}

	// step 1 group b
	i = suffixPos(s, []rune("en"))
	if i == -1 {
		i = suffixPos(s, []rune("es"))
	}
	if i == -1 {
		i = suffixPos(s, []rune("e"))
	}
	if i >= r1 {
		s = s[:i]
		if x := suffixPos(s, []rune("niss")); x != -1 {
			s = s[:len(s)-1]
		}
		goto step2
	}

	// step 1 group c
	if i = suffixPos(s, []rune("s")); i >= r1 && hasValidSEnding(s[:len(s)-1]) { // only delete if preceded by valid s-ending
		s = s[:i]
		goto step2
	}

step2:
	// step 2 group a
	i = suffixPos(s, []rune("est"))
	if i == -1 {
		i = suffixPos(s, []rune("en"))
	}
	if i == -1 {
		i = suffixPos(s, []rune("er"))
	}
	if i >= r1 {
		s = s[:i]
		goto step3
	}

	// step 2 group b
	// st-ending itself must be preceded by at least 3 letters -> min 3 + st-end 1 + st 2 = 6 
	if i = suffixPos(s, []rune("st")); i >= r1 && hasValidStEnding(s[:len(s)-2]) && len(s) >= 6 {
		s = s[:i]
		goto step3
	}

step3:
	// step 3 end ung
	i = suffixPos(s, []rune("end"))
	if i == -1 {
		i = suffixPos(s, []rune("ung"))
	}
	if i >= r2 {
		s = s[:i]
		// if preceded by ig, delete if in R2 and not preceded by e
		if i = suffixPos(s, []rune("ig")); i >= r2 && suffixPos(s[:i], []rune("e")) == -1 {
			s = s[:i]
		}
		goto final
	}

	// step 3 ig ik isch
	i = suffixPos(s, []rune("ig"))
	if i == -1 {
		i = suffixPos(s, []rune("ik"))
	}
	if i == -1 {
		i = suffixPos(s, []rune("isch"))
	}
	if i >= r2 && suffixPos(s[:i], []rune("e")) == -1 {
		s = s[:i]
		goto final
	}

	// step 3 lich heit
	i = suffixPos(s, []rune("lich"))
	if i == -1 {
		i = suffixPos(s, []rune("heit"))
	}
	if i >= r2 {
		s = s[:i]
		// if preceded by er or en, delete if in R1 
		i = suffixPos(s, []rune("er"))
		if i == -1 {
			i = suffixPos(s, []rune("en"))
		}
		if i >= r1 {
			s = s[:i]
		}
		goto final
	}

	// step 3 keit
	i = suffixPos(s, []rune("keit"))
	if i >= r2 {
		s = s[:i]
		// if preceded by lich or ig, delete if in R2 
		i = suffixPos(s, []rune("lich"))
		if i == -1 {
			i = suffixPos(s, []rune("ig"))
		}
		if i >= r2 {
			s = s[:i]
		}
		goto final
	}

final:
	for i, max := 0, len(s); i < max; i++ {
		switch s[i] {
		case 'U':
			s[i] = 'u'
		case 'Y':
			s[i] = 'y'
		case 'ä':
			s[i] = 'a'
		case 'ö':
			s[i] = 'o'
		case 'ü':
			s[i] = 'u'
		}
	}

	return string(s)
}
