package main

import (
	"bufio"
	"database/sql"
	"errors"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/jbrukh/bayesian"
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
	log.Info("training classifier")
	if err := trainClassifier(c, s); err != nil {
		return err
	}
	if err := c.WriteToFile(path.Join("data", "bayes.dat")); err != nil {
		return err
	}
	return nil
}

func loadClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	log.Debug("loading classifier")
	filename := path.Join("data", "bayes.dat")
	var err error
	c, err = bayesian.NewClassifierFromFile(filename)
	if err != nil && err.Error() == "open data/bayes.dat: no such file or directory" {
		log.Warn("classifier file not found. building...")
		c, err = buildClassifier(c)
		if err != nil {
			return c, err
		}
	} else if err != nil {
		log.Info("error loading bayes.dat", err)
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
			log.Error("line", line, "::", err)
		}
		line++
	}
	if err = scanner.Err(); err != nil {
		return c, err
	}
	if err = c.WriteToFile(path.Join("data", "bayes.dat")); err != nil {
		return c, err
	}
	log.Debug("new classifier trained")
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

// addContext to a StructuredInput, adding a user identifier and replacing
// pronouns with the nouns to which they refer.
func addContext(si *datatypes.StructuredInput, uid int, fid string, fidT int) (
	*datatypes.StructuredInput, bool, error) {
	u, err := getUser(si)
	if err != nil && err != sql.ErrNoRows {
		return si, false, err
	}
	ctxAdded := false
	for _, w := range si.Pronouns() {
		var ctx string
		switch datatypes.Pronouns[w] {
		case datatypes.ObjectI:
			ctx, err = getContextObject(u, si, "objects")
			if err != nil {
				return si, false, err
			}
			for i, o := range si.Objects {
				if o != w {
					continue
				}
				si.Objects[i] = ctx
				ctxAdded = true
			}
		case datatypes.ActorI:
			ctx, err = getContextObject(u, si, "actors")
			if err != nil {
				return si, false, err
			}
			for i, o := range si.Actors {
				if o != w {
					continue
				}
				si.Actors[i] = ctx
				ctxAdded = true
			}
		case datatypes.TimeI:
			ctx, err = getContextObject(u, si, "times")
			if err != nil {
				return si, false, err
			}
			for i, o := range si.Times {
				if o != w {
					continue
				}
				si.Times[i] = ctx
				ctxAdded = true
			}
		case datatypes.PlaceI:
			ctx, err = getContextObject(u, si, "places")
			if err != nil {
				return si, false, err
			}
			for i, o := range si.Places {
				if o != w {
					continue
				}
				si.Places[i] = ctx
				ctxAdded = true
			}
		default:
			return si, false, errors.New("unknown type found for pronoun")
		}
		log.Debug("ctx: ", ctx)
	}
	return si, ctxAdded, nil
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
	if m <= 0.7 {
		log.Debug(word1, " || ", datatypes.String[likely+1])
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
