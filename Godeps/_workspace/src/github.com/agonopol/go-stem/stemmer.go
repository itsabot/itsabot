package stemmer

import "fmt"
import "bytes"

func ingore() {
	fmt.Sprintf("")
}

func Consonant(body []byte, offset int) bool {
	switch body[offset] {
	case 'A', 'E', 'I', 'O', 'U', 'a', 'e', 'i', 'o', 'u':
		return false
	case 'Y', 'y':
		if offset == 0 {
			return true
		}
		return offset > 0 && !Consonant(body, offset-1)
	}
	return true
}

func Vowel(body []byte, offset int) bool {
	return !Consonant(body, offset)
}

const (
	vowel_state = iota
	consonant_state
)

func Measure(body []byte) int {
	meansure := 0
	if len(body) > 0 {
		var state int
		if Vowel(body, 0) {
			state = vowel_state
		} else {
			state = consonant_state
		}
		for i := 0; i < len(body); i++ {
			if Vowel(body, i) && state == consonant_state {
				state = vowel_state
			} else if Consonant(body, i) && state == vowel_state {
				state = consonant_state
				meansure++
			}
		}
	}
	return meansure
}

func hasVowel(body []byte) bool {
	for i := 0; i < len(body); i++ {
		if Vowel(body, i) {
			return true
		}
	}
	return false
}

func one_a(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("sses")) || bytes.HasSuffix(body, []byte("ies")) {
		return body[:len(body)-2]
	} else if bytes.HasSuffix(body, []byte("ss")) {
		return body
	} else if bytes.HasSuffix(body, []byte("s")) {
		return body[:len(body)-1]
	}
	return body
}

func star_o(body []byte) bool {
	size := len(body) - 1
	if size >= 2 && Consonant(body, size-2) && Vowel(body, size-1) && Consonant(body, size) {
		return body[size] != 'w' && body[size] != 'x' && body[size] != 'y'
	}
	return false
}
func one_b_a(body []byte) []byte {

	size := len(body)
	if bytes.HasSuffix(body, []byte("at")) {
		return append(body, 'e')
	} else if bytes.HasSuffix(body, []byte("bl")) {
		return append(body, 'e')
	} else if bytes.HasSuffix(body, []byte("iz")) {
		return append(body, 'e')
	} else if Consonant(body, size-1) && Consonant(body, size-2) && body[size-1] == body[size-2] {
		if body[size-1] != 'l' && body[size-1] != 's' && body[size-1] != 'z' {
			return body[:size-1]
		}
	} else if star_o(body) && Measure(body) == 1 {
		return append(body, 'e')
	}
	return body
}

func one_b(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("eed")) {
		if Measure(body[:len(body)-3]) > 0 {
			return body[:len(body)-1]
		}
	} else if bytes.HasSuffix(body, []byte("ed")) {
		if hasVowel(body[:len(body)-2]) {
			return one_b_a(body[:len(body)-2])
		}
	} else if bytes.HasSuffix(body, []byte("ing")) {
		if hasVowel(body[:len(body)-3]) {
			return one_b_a(body[:len(body)-3])
		}
	}
	return body
}

func one_c(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("y")) && hasVowel(body[:len(body)-1]) {
		body[len(body)-1] = 'i'
		return body
	}
	return body
}

func two(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("ational")) {
		if Measure(body[:len(body)-7]) > 0 {
			return append(body[:len(body)-7], []byte("ate")...)
		}
	} else if bytes.HasSuffix(body, []byte("tional")) {
		if Measure(body[:len(body)-6]) > 0 {
			return body[:len(body)-2]
		}
	} else if bytes.HasSuffix(body, []byte("enci")) || bytes.HasSuffix(body, []byte("anci")) {
		if Measure(body[:len(body)-4]) > 0 {
			return append(body[:len(body)-1], 'e')
		}
	} else if bytes.HasSuffix(body, []byte("izer")) {
		if Measure(body[:len(body)-4]) > 0 {
			return append(body[:len(body)-4], []byte("ize")...)
		}
	} else if bytes.HasSuffix(body, []byte("abli")) {
		if Measure(body[:len(body)-4]) > 0 {
			return append(body[:len(body)-4], []byte("able")...)
		}
	// To match the published algorithm, delete the following phrase
	} else if bytes.HasSuffix(body, []byte("bli")) {
		if Measure(body[:len(body)-3]) > 0 {
			return append(body[:len(body)-1], 'e')
		}
	} else if bytes.HasSuffix(body, []byte("alli")) {
		if Measure(body[:len(body)-4]) > 0 {
			return append(body[:len(body)-4], []byte("al")...)
		}
	} else if bytes.HasSuffix(body, []byte("entli")) {
		if Measure(body[:len(body)-5]) > 0 {
			return append(body[:len(body)-5], []byte("ent")...)
		}
	} else if bytes.HasSuffix(body, []byte("eli")) {
		if Measure(body[:len(body)-3]) > 0 {
			return append(body[:len(body)-3], []byte("e")...)
		}
	} else if bytes.HasSuffix(body, []byte("ousli")) {
		if Measure(body[:len(body)-5]) > 0 {
			return append(body[:len(body)-5], []byte("ous")...)
		}
	} else if bytes.HasSuffix(body, []byte("ization")) {
		if Measure(body[:len(body)-7]) > 0 {
			return append(body[:len(body)-7], []byte("ize")...)
		}
	} else if bytes.HasSuffix(body, []byte("ation")) {
		if Measure(body[:len(body)-5]) > 0 {
			return append(body[:len(body)-5], []byte("ate")...)
		}
	} else if bytes.HasSuffix(body, []byte("ator")) {
		if Measure(body[:len(body)-4]) > 0 {
			return append(body[:len(body)-4], []byte("ate")...)
		}
	} else if bytes.HasSuffix(body, []byte("alism")) {
		if Measure(body[:len(body)-5]) > 0 {
			return append(body[:len(body)-5], []byte("al")...)
		}
	} else if bytes.HasSuffix(body, []byte("iveness")) {
		if Measure(body[:len(body)-7]) > 0 {
			return append(body[:len(body)-7], []byte("ive")...)
		}
	} else if bytes.HasSuffix(body, []byte("fulness")) {
		if Measure(body[:len(body)-7]) > 0 {
			return append(body[:len(body)-7], []byte("ful")...)
		}
	} else if bytes.HasSuffix(body, []byte("ousness")) {
		if Measure(body[:len(body)-7]) > 0 {
			return append(body[:len(body)-7], []byte("ous")...)
		}
	} else if bytes.HasSuffix(body, []byte("aliti")) {
		if Measure(body[:len(body)-5]) > 0 {
			return append(body[:len(body)-5], []byte("al")...)
		}
	} else if bytes.HasSuffix(body, []byte("iviti")) {
		if Measure(body[:len(body)-5]) > 0 {
			return append(body[:len(body)-5], []byte("ive")...)
		}
	} else if bytes.HasSuffix(body, []byte("biliti")) {
		if Measure(body[:len(body)-6]) > 0 {
			return append(body[:len(body)-6], []byte("ble")...)
		}
	// To match the published algorithm, delete the following phrase
	} else if bytes.HasSuffix(body, []byte("logi")) {
		if Measure(body[:len(body)-4]) > 0 {
			return body[:len(body)-1]
		}
	}
	return body
}

func three(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("icate")) {
		if Measure(body[:len(body)-5]) > 0 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ative")) {
		if Measure(body[:len(body)-5]) > 0 {
			return body[:len(body)-5]
		}
	} else if bytes.HasSuffix(body, []byte("alize")) {
		if Measure(body[:len(body)-5]) > 0 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("iciti")) {
		if Measure(body[:len(body)-5]) > 0 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ical")) {
		if Measure(body[:len(body)-4]) > 0 {
			return body[:len(body)-2]
		}
	} else if bytes.HasSuffix(body, []byte("ful")) {
		if Measure(body[:len(body)-3]) > 0 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ness")) {
		if Measure(body[:len(body)-4]) > 0 {
			return body[:len(body)-4]
		}
	}
	return body
}

func four(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("al")) {
		if Measure(body[:len(body)-2]) > 1 {
			return body[:len(body)-2]
		}
	} else if bytes.HasSuffix(body, []byte("ance")) {
		if Measure(body[:len(body)-4]) > 1 {
			return body[:len(body)-4]
		}
	} else if bytes.HasSuffix(body, []byte("ence")) {
		if Measure(body[:len(body)-4]) > 1 {
			return body[:len(body)-4]
		}
	} else if bytes.HasSuffix(body, []byte("er")) {
		if Measure(body[:len(body)-2]) > 1 {
			return body[:len(body)-2]
		}
	} else if bytes.HasSuffix(body, []byte("ic")) {
		if Measure(body[:len(body)-2]) > 1 {
			return body[:len(body)-2]
		}
	} else if bytes.HasSuffix(body, []byte("able")) {
		if Measure(body[:len(body)-4]) > 1 {
			return body[:len(body)-4]
		}
	} else if bytes.HasSuffix(body, []byte("ible")) {
		if Measure(body[:len(body)-4]) > 1 {
			return body[:len(body)-4]
		}
	} else if bytes.HasSuffix(body, []byte("ant")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ement")) {
		if Measure(body[:len(body)-5]) > 1 {
			return body[:len(body)-5]
		}
	} else if bytes.HasSuffix(body, []byte("ment")) {
		if Measure(body[:len(body)-4]) > 1 {
			return body[:len(body)-4]
		}
	} else if bytes.HasSuffix(body, []byte("ent")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ion")) {
		if Measure(body[:len(body)-3]) > 1 {
			if len(body) > 4 && (body[len(body)-4] == 's' || body[len(body)-4] == 't') {
				return body[:len(body)-3]
			}
		}
	} else if bytes.HasSuffix(body, []byte("ou")) {
		if Measure(body[:len(body)-2]) > 1 {
			return body[:len(body)-2]
		}
	} else if bytes.HasSuffix(body, []byte("ism")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ate")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("iti")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ous")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ive")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	} else if bytes.HasSuffix(body, []byte("ize")) {
		if Measure(body[:len(body)-3]) > 1 {
			return body[:len(body)-3]
		}
	}
	return body
}

func five_a(body []byte) []byte {
	if bytes.HasSuffix(body, []byte("e")) && Measure(body[:len(body)-1]) > 1 {
		return body[:len(body)-1]
	} else if bytes.HasSuffix(body, []byte("e")) && Measure(body[:len(body)-1]) == 1 && !star_o(body[:len(body)-1]) {
		return body[:len(body)-1]
	}
	return body
}

func five_b(body []byte) []byte {
	size := len(body)
	if Measure(body) > 1 && Consonant(body, size-1) && Consonant(body, size-2) && body[size-1] == body[size-2] && body[size-1] == 'l' {
		return body[:len(body)-1]
	}
	return body
}

func Stem(body []byte) []byte {
	word := bytes.TrimSpace(bytes.ToLower(body))
	if len(word) > 2 {
		return five_b(five_a(four(three(two(one_c(one_b(one_a(word))))))))
	}
	return word
}
