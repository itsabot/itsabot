package language

import (
	"encoding/json"
	"math/rand"
	"regexp"
	"sort"
	"strings"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/mattbaird/elastigo/lib"
)

var regexArticle = regexp.MustCompile(`^(a|an|the|A|An|The)\s`)

type WordT struct {
	Word  string
	POS   string
	Index int
}

type ByIndex []WordT

func (a ByIndex) Len() int {
	return len(a)
}

func (a ByIndex) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByIndex) Less(i, j int) bool {
	return a[i].Index < a[j].Index
}

func (a ByIndex) StringSlice() []string {
	var tmp []string
	for _, el := range a {
		tmp = append(tmp, el.Word)
	}
	return tmp
}

// Summarize identifies keyword phrases in text. keywordSource is the
// ElasticSearch type in the form of index_type. For example, to identify
// keywords in a wine review, keywordSource would be "products_alcohol".
func Summarize(product *dt.Product, keywordSource string) (string, error) {
	// TODO catch negative connations in a clause, so the summary does not
	// include or emphasize them.
	shortSummary := buildShortSummary(product)
	if len(product.Reviews) == 0 {
		return shortSummary, nil
	}
	text := product.Reviews[0].Body
	ec := dt.NewSearchClient()
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]string{"Name": text},
		},
		"size": 50,
	}
	res, err := ec.Search("keywords", keywordSource, nil, q)
	if err != nil {
		return shortSummary, err
	}
	keywords, err := extractKeywords(text, res.Hits.Hits)
	if err != nil {
		return shortSummary, err
	}
	keywords = combineKeywordsIntoRanges(keywords)
	summary := buildSummary(product, keywords)
	return shortSummary + summary, nil
}

func buildSummary(product *dt.Product, keywords []WordT) string {
	var totalNounLen int
	var totalAdjLen int
	var summary string
	var nouns []string
	var adjs []string
	nounAndOr := "and"
	adjAndOr := "and"
	for _, wt := range keywords {
		if wt.POS == "n" {
			nouns = append(nouns, wt.Word)
			totalNounLen += len(wt.Word)
			if strings.Contains(wt.Word, "and") {
				nounAndOr = "as well as"
			}
		} else if wt.POS == "adj" {
			adjs = append(adjs, wt.Word)
			totalAdjLen += len(wt.Word)
			if strings.Contains(wt.Word, "and") {
				adjAndOr = "as well as"
			}
		}
	}
	var rem string
	var addition string
	if totalNounLen >= totalAdjLen {
		n := rand.Intn(2)
		switch n {
		case 0:
			summary = "hints of "
		case 1:
			summary = "notes of "
		}
		n = rand.Intn(10)
		switch n {
		case 0:
			summary = "It features " + summary
		case 1:
			summary = "It's characterized by " + summary
		case 2:
			summary = "It's known for " + summary
		case 3:
			summary = "It's known for its " + summary
		case 4:
			summary = "It's known for having " + summary
		case 5:
			summary = "It has " + summary
		case 6:
			summary = "You'll experience " + summary
		case 7:
			summary = "You'll sense " + summary
		case 8:
			summary = "You'll enjoy " + summary
		case 9:
			summary = "You'll love its " + summary
		}
		for i := 0; i <= len(nouns)-1; i++ {
			// TODO design more robust maxLength control
			if totalNounLen > 60 {
				rem, nouns = nouns[len(nouns)-1], nouns[:len(nouns)-1]
				totalNounLen -= len(rem)
			}
		}
		for _, noun := range nouns {
			ws := strings.Fields(noun)
			for _, w := range ws {
				if w == "is" {
					nounAndOr = "."
					break
				}
			}
		}
		addition = SliceToString(nouns, nounAndOr)
	} else {
		summary = "It's "
		for i := 0; i <= len(adjs)-1; i++ {
			if totalAdjLen > 60 {
				rem, adjs = adjs[len(adjs)-1], adjs[:len(adjs)-1]
				totalAdjLen -= len(rem)
			}
		}
		for _, adj := range adjs {
			ws := strings.Fields(adj)
			for _, w := range ws {
				if w == "is" {
					adjAndOr = "."
					break
				}
			}
		}
		addition = SliceToString(adjs, adjAndOr)
	}
	if len(addition) == 0 {
		return ""
	}
	summary += strings.TrimRight(addition, ".,;:!?'\"") + "."
	return summary
}

func combineKeywordsIntoRanges(keywords []WordT) []WordT {
	var ranges []WordT
	var buf *WordT
	for i := 0; i < len(keywords); i++ {
		buf, i = appendStops(buf, keywords, i, 1)
		if buf == nil {
			if i == len(keywords) {
				i--
			}
			buf = &keywords[i]
		}
		buf.Word = strings.TrimRight(buf.Word, ".,;:'\"?!")
		ranges = append(ranges, *buf)
		buf = nil
	}
	return ranges
}

func appendStops(buf *WordT, keywords []WordT, i, j int) (*WordT, int) {
	kw := keywords[i]
	// prevent joining initial stopwords
	if Contains(SummaryStopWords, kw.Word) {
		return buf, i
	}
	// if next word is also a keyword
	if i < len(keywords)-j && keywords[i+j].Index == kw.Index+j {
		if buf == nil {
			buf = &WordT{
				Index: kw.Index,
				POS:   keywords[i+j].POS,
				Word:  kw.Word + " " + keywords[i+j].Word,
			}
		} else {
			buf.POS = keywords[i+j].POS
			buf.Word += " " + keywords[i+j].Word
		}
		// separated by a period. don't continue
		lastLetter := keywords[i+j].Word[len(keywords[i+j].Word)-1]
		if lastLetter == '.' || lastLetter == ';' {
			return buf, i + 1
		}
		buf, i = appendStops(buf, keywords, i, j+1)
	}
	return buf, i + 1
}

func extractKeywords(text string, hits []elastigo.Hit) ([]WordT, error) {
	var keywords []WordT
	words := strings.Fields(text)
	for i := 0; i < len(words); i++ {
		word := words[i]
		if Contains(SummaryStopWords, word) {
			keywords = append(keywords, WordT{
				Word:  word,
				Index: i,
			})
			continue
		}
	}
	for _, hit := range hits {
		for i := 0; i < len(words); i++ {
			word := words[i]
			tmp := strings.TrimRight(word, ".,;:?!'\"")
			if tmp == hit.Id {
				var tmp WordT
				bytes, err := hit.Source.MarshalJSON()
				if err != nil {
					return keywords, err
				}
				if err = json.Unmarshal(bytes, &tmp); err != nil {
					return keywords, err
				}
				tmp.Word = word
				tmp.Index = i
				keywords = append(keywords, tmp)
			}
		}
	}
	sort.Sort(ByIndex(keywords))
	return keywords, nil
}

var SummaryStopWords = []string{
	"and",
	"but",
	"for",
	"a",
	"an",
	"the",
	"yet",
	"with",
	"so",
	"of",
	"by",
	"is",
	"before",
	"after",
	"above",
	"below",
	"over",
	"under",
	"some",
	"alongside",
}

func Contains(wordList []string, s string) bool {
	s = strings.TrimRight(strings.ToLower(s), ".,;:!?'\"")
	for _, word := range wordList {
		if s == word {
			return true
		}
	}
	return false
}

func buildShortSummary(p *dt.Product) string {
	if len(p.Category) == 0 {
		return ""
	}
	tmp := "It's "
	n := rand.Intn(8)
	switch n {
	case 0:
		tmp += "a gorgeous "
	case 1:
		tmp += "a brilliant "
	case 2:
		tmp += "an amazing "
	case 3:
		tmp += "a spectacular "
	case 4:
		tmp += "a tasty "
	case 5:
		tmp += "a delicious "
	case 6, 7:
		tmp += "a "
	}
	return tmp + p.Category + ". "
}
