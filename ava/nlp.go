package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"log"
	"os"
	"path"
	"strings"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jbrukh/bayesian"
	"github.com/avabot/ava/shared/datatypes"
)

const (
	Command bayesian.Class = "Command"
	Actor   bayesian.Class = "Actor"
	Object  bayesian.Class = "Object"
	Time    bayesian.Class = "Time"
	Place   bayesian.Class = "Place"
	None    bayesian.Class = "None"
)

var (
	ErrMissingFlexIdType = errors.New("missing flexidtype")
	ErrSentenceTooShort  = errors.New("sentence too short to classify")
)

func train(c *bayesian.Classifier, s string) error {
	log.Println("training classifier")
	if err := trainClassifier(c, s); err != nil {
		return err
	}
	buf := bytes.NewBuffer([]byte{})
	if err := c.WriteTo(buf); err != nil {
		return err
	}
	q := `UPDATE ml SET data=$1 WHERE name='ner'`
	_, err := db.Exec(q, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func loadClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	log.Println("loading classifier")
	var err error
	q := `SELECT data FROM ml WHERE name='ner' LIMIT 1`
	row := db.QueryRowx(q)
	var tmp []byte
	err = row.Scan(&tmp)
	buf := bytes.NewBuffer(tmp)
	if err == sql.ErrNoRows {
		c, err = buildClassifier(c)
		if err != nil {
			log.Println("err building classifier")
			return c, err
		}
	} else if err != nil {
		log.Println("err getting classifier from DB")
		return c, err
	}
	c, err = bayesian.NewClassifierFromReader(buf)
	if err != nil {
		log.Println("err building new classifier from reader", err)
		return c, err
	}
	return c, nil
}

func buildClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	c = bayesian.NewClassifier(Command, Actor, Object, Time, Place, None)
	filename := path.Join("data", "training", "imperative.txt")
	fi, err := os.Open(filename)
	if err != nil {
		return c, err
	}
	defer fi.Close()
	scanner := bufio.NewScanner(fi)
	line := 1
	for scanner.Scan() {
		if err := trainClassifier(c, scanner.Text()); err != nil {
			log.Println("err: line", line, "::", err)
		}
		line++
	}
	if err := scanner.Err(); err != nil {
		return c, err
	}
	buf := bytes.NewBuffer([]byte{})
	if err := c.WriteTo(buf); err != nil {
		return c, err
	}
	q := `INSERT INTO ml (name, data) VALUES ('ner', $1)`
	_, err = db.Exec(q, buf.Bytes())
	if err != nil {
		log.Println("err updating ml.ner", err)
		return c, err
	}
	log.Println("new classifier trained")
	return c, nil
}

func trainClassifier(c *bayesian.Classifier, s string) error {
	if len(s) == 0 || s[0] == '/' {
		return nil
	}
	ws := strings.Fields(s)
	l := len(ws)
	for i := 0; i < l; i++ {
		var word2 string
		var word3 string
		word1, entity, err := extractEntity(ws[i])
		if err != nil {
			return err
		}
		if entity == "" {
			continue
		}
		trigram := word1
		if i+1 < l {
			word2, _, err = extractEntity(ws[i+1])
			if err != nil {
				return err
			}
			trigram += " " + word2
		}
		if i+2 < l {
			word3, _, err = extractEntity(ws[i+2])
			if err != nil {
				return err
			}
			trigram += " " + word3
		}
		c.Learn([]string{word1}, entity)
		if word2 != "" {
			c.Learn([]string{word1 + " " + word2}, entity)
		}
		if word3 != "" {
			c.Learn([]string{trigram}, entity)
		}
	}
	return nil
}

// classify builds a StructuredInput from a sentence. The bool specifies whether
// additional training is needed for that sentence.
func classify(c *bayesian.Classifier, s string) (*datatypes.StructuredInput, bool, error) {
	si := &datatypes.StructuredInput{}
	if len(s) == 0 {
		return si, false, ErrSentenceTooShort
	}
	ws := strings.Fields(s)
	var wc []datatypes.WordClass
	var needsTraining bool
	for i := range ws {
		var err error
		var tmp datatypes.WordClass
		tmp, needsTraining, err = classifyTrigram(c, s, ws, i)
		if err != nil {
			return si, false, err
		}
		wc = append(wc, tmp)
	}
	if err := si.Add(wc); err != nil {
		return si, false, err
	}
	return si, needsTraining, nil
}

// addContext to a StructuredInput, replacing pronouns with the nouns to which
// they refer. TODO refactor
func addContext(m *datatypes.Message) (*datatypes.Message, bool, error) {
	ctxAdded := false
	for _, w := range m.Input.StructuredInput.Pronouns() {
		var ctx string
		var err error
		switch datatypes.Pronouns[w] {
		case datatypes.ObjectI:
			ctx, err = getContextObject(m.User,
				m.Input.StructuredInput,
				"objects")
			if err != nil {
				return m, false, err
			}
			if ctx == "" {
				return m, false, nil
			}
			for i, o := range m.Input.StructuredInput.Objects {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Objects[i] = ctx
				ctxAdded = true
			}
		case datatypes.ActorI:
			ctx, err = getContextObject(m.User, m.Input.StructuredInput,
				"actors")
			if err != nil {
				return m, false, err
			}
			if ctx == "" {
				return m, false, nil
			}
			for i, o := range m.Input.StructuredInput.Actors {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Actors[i] = ctx
				ctxAdded = true
			}
		case datatypes.TimeI:
			ctx, err = getContextObject(m.User, m.Input.StructuredInput,
				"times")
			if err != nil {
				return m, false, err
			}
			if ctx == "" {
				return m, false, nil
			}
			for i, o := range m.Input.StructuredInput.Times {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Times[i] = ctx
				ctxAdded = true
			}
		case datatypes.PlaceI:
			ctx, err = getContextObject(m.User, m.Input.StructuredInput,
				"places")
			if err != nil {
				return m, false, err
			}
			if ctx == "" {
				return m, false, nil
			}
			for i, o := range m.Input.StructuredInput.Places {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Places[i] = ctx
				ctxAdded = true
			}
		default:
			return m, false,
				errors.New("unknown type found for pronoun")
		}
		log.Println("ctx: ", ctx)
	}
	return m, ctxAdded, nil
}

// extractEntity from a word. If a Command, strip any contraction. For example,
// "where's" -> where. Since Ava ignores linking verbs, there's no need to
// add "is" back into the sentence.
func extractEntity(w string) (string, bayesian.Class, error) {
	w = strings.TrimRight(w, ").,;?!:")
	if w[0] != '_' {
		return w, "", nil
	}
	switch w[1] {
	case 'C':
		return w[3:], Command, nil
	case 'O':
		return w[3:], Object, nil
	case 'A':
		return w[3:], Actor, nil
	case 'T':
		return w[3:], Time, nil
	case 'P':
		return w[3:], Place, nil
	case 'N':
		return w[3:], None, nil
	}
	return w, "", errors.New("syntax error in entity")
}

// classifyTrigram determines the best classification for a word in a sentence
// given its surrounding context (i, i+1, i+2). Underflow on the returned
// probabilities is possible, but ignored, since classifyTrigram prefers a >=70%
// confidence level. The bool returned specified whether additional training is
// needed.
func classifyTrigram(c *bayesian.Classifier, s string, ws []string, i int) (
	datatypes.WordClass, bool, error) {
	// TODO: Given the last 2 words of a sentence, construct the trigram
	// including prior words.
	var wc datatypes.WordClass
	l := len(ws)
	word1, _, err := extractEntity(ws[i])
	if err != nil {
		return wc, false, err
	}
	bigram := word1
	trigram := word1
	var word2, word3 string
	if i+1 < l {
		word2, _, err = extractEntity(ws[i+1])
		if err != nil {
			return wc, false, err
		}
		bigram += " " + word2
		trigram += " " + word2
	}
	if i+2 < l {
		word3, _, err = extractEntity(ws[i+2])
		if err != nil {
			return wc, false, err
		}
		trigram += " " + word3
	}
	var needsTraining bool
	probs, likely, _ := c.ProbScores([]string{trigram})
	if max(probs) <= 0.7 {
		probs, likely, _ = c.ProbScores([]string{bigram})
		needsTraining = true
	}
	m := max(probs)
	if m <= 0.7 {
		probs, likely, _ = c.ProbScores([]string{word1})
		needsTraining = true
	}
	m = max(probs)
	if m <= 0.7 {
		log.Println(word1, " || ", datatypes.String[likely+1], " || ", m)
		needsTraining = true
	}
	return datatypes.WordClass{word1, likely + 1}, needsTraining, nil
}

func max(slice []float64) float64 {
	m := slice[0]
	for index := 1; index < len(slice); index++ {
		if slice[index] > m {
			m = slice[index]
		}
	}
	return m
}
