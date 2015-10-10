package main

import (
	"bufio"
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
	if err := c.WriteToFile(path.Join("data", "bayes.dat")); err != nil {
		return err
	}
	return nil
}

func loadClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	log.Println("loading classifier")
	filename := path.Join("data", "bayes.dat")
	var err error
	c, err = bayesian.NewClassifierFromFile(filename)
	if err != nil && err.Error() == "open data/bayes.dat: no such file or directory" {
		log.Println("warn: classifier file not found. building...")
		c, err = buildClassifier(c)
		if err != nil {
			return c, err
		}
	} else if err != nil {
		log.Println("error loading bayes.dat", err)
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
	if err = scanner.Err(); err != nil {
		return c, err
	}
	if err = c.WriteToFile(path.Join("data", "bayes.dat")); err != nil {
		return c, err
	}
	log.Println("new classifier trained")
	return c, nil
}

func trainClassifier(c *bayesian.Classifier, s string) error {
	if len(s) == 0 {
		return ErrSentenceTooShort
	}
	if s[0] == '/' {
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

func classify(c *bayesian.Classifier, s string) (*datatypes.StructuredInput, error) {
	si := &datatypes.StructuredInput{}
	if len(s) == 0 {
		return si, ErrSentenceTooShort
	}
	ws := strings.Fields(s)
	var wc []datatypes.WordClass
	for i := range ws {
		tmp, err := classifyTrigram(c, ws, i)
		if err != nil {
			return si, err
		}
		wc = append(wc, tmp)
	}
	if err := si.Add(wc); err != nil {
		return si, err
	}
	return si, nil
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
	w = strings.TrimRight(w, ").,;?")
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
// confidence level.
func classifyTrigram(c *bayesian.Classifier, ws []string, i int) (
	datatypes.WordClass, error) {
	// TODO: Given the last 2 words of a sentence, construct the trigram
	// including prior words.
	var wc datatypes.WordClass
	l := len(ws)
	word1, _, err := extractEntity(ws[i])
	if err != nil {
		return wc, err
	}
	word1c := stripContraction(word1)
	bigram := word1c
	trigram := word1c
	var word2, word2c, word3, word3c string
	if i+1 < l {
		word2, _, err = extractEntity(ws[i+1])
		if err != nil {
			return wc, err
		}
		word2c = stripContraction(word2)
		bigram += " " + word2c
		trigram += " " + word2c
	}
	if i+2 < l {
		word3, _, err = extractEntity(ws[i+2])
		if err != nil {
			return wc, err
		}
		word3c = stripContraction(word3)
		trigram += " " + word3c
	}
	probs, likely, _ := c.ProbScores([]string{trigram})
	if max(probs) <= 0.7 {
		probs, likely, _ = c.ProbScores([]string{bigram})
	}
	m := max(probs)
	if m <= 0.7 {
		probs, likely, _ = c.ProbScores([]string{word1})
	}
	// TODO design a process for automated training when confidence remains
	// low.
	m = max(probs)
	if m <= 0.7 {
		log.Println(word1, " || ", datatypes.String[likely+1], " || ", m)
	}
	return datatypes.WordClass{word1, likely + 1}, nil
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

func stripContraction(w string) string {
	// TODO Check contractions.txt for reasonable things that should be
	// added back.
	if len(w) <= 2 {
		return w
	}
	if w[len(w)-2] == '\'' {
		return w[:len(w)-2]
	}
	if w[len(w)-3] == '\'' {
		return w[:len(w)-3]
	}
	return w
}
