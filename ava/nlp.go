package main

import (
	"bufio"
	"errors"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

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
		return nil
	}
	if s[0] == '/' {
		return nil
	}
	ws, err := extractFields(s)
	if err != nil {
		return err
	}
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

func extractFields(s string) ([]string, error) {
	var ss []string
	if len(s) == 0 {
		return ss, errors.New("sentence too short to classify")
	}
	wordBuf := ""
	ws := strings.Fields(s)
	for _, w := range ws {
		r, _ := utf8.DecodeRuneInString(w)
		if r == '_' {
			r, _ = utf8.DecodeRuneInString(w[3:])
		}
		if unicode.IsNumber(r) {
			wordBuf += w + " "
			continue
		}
		word, _, err := extractEntity(w)
		if err != nil {
			return ss, err
		}
		switch strings.ToLower(word) {
		// Articles and prepositions
		case "a", "an", "the", "before", "at", "after", "next", "to":
			wordBuf += w + " "
		default:
			ss = append(ss, wordBuf+w)
			wordBuf = ""
		}
	}
	return ss, nil
}

func classify(c *bayesian.Classifier, s string) (*datatypes.StructuredInput, error) {
	si := &datatypes.StructuredInput{}
	ws, err := extractFields(s)
	if err != nil {
		return si, err
	}
	var wc []datatypes.WordClass
	for i := range ws {
		tmp, err := classifyTrigram(c, ws, i)
		if err != nil {
			return si, err
		}
		wc = append(wc, tmp)
	}
	if err = si.Add(wc); err != nil {
		return si, err
	}
	log.Info(si.String())
	return si, nil
}

func extractEntity(w string) (string, bayesian.Class, error) {
	w = strings.TrimRight(w, ").,;")
	if w[0] != '_' {
		return w, "", nil
	}
	switch w[1] {
	case 'C': // Command
		return w[3:], Command, nil
	case 'O': // Object
		return w[3:], Object, nil
	case 'A': // Actor
		return w[3:], Actor, nil
	case 'T': // Time
		return w[3:], Time, nil
	case 'P':
		return w[3:], Place, nil
	case 'N': // None
		return w[3:], None, nil
	}
	return w, "", errors.New("syntax error in entity")
}

func classifyTrigram(c *bayesian.Classifier, ws []string, i int) (datatypes.WordClass,
	error) {

	var wc datatypes.WordClass
	l := len(ws)
	word1, _, err := extractEntity(ws[i])
	if err != nil {
		return wc, err
	}
	log.Debug("word: ", word1)
	bigram := word1
	trigram := word1
	var word2 string
	var word3 string
	if i+1 < l {
		word2, _, err = extractEntity(ws[i+1])
		if err != nil {
			return wc, err
		}
		bigram += " " + word2
		trigram += " " + word2
	}
	if i+2 < l {
		word3, _, err = extractEntity(ws[i+2])
		if err != nil {
			return wc, err
		}
		trigram += " " + word3
	}
	probs, likely, _, err := c.SafeProbScores([]string{trigram})
	if err != nil {
		return wc, err
	}
	if max(probs) <= 0.7 {
		log.Debug("try 2")
		probs, likely, _, err = c.SafeProbScores([]string{bigram})
		if err != nil {
			return wc, err
		}
	}
	if max(probs) <= 0.7 {
		log.Debug("try 3")
		probs, likely, _, err = c.SafeProbScores([]string{word1})
		if err != nil {
			return wc, err
		}
	}
	// TODO Design a process for automated training
	log.Info(probs)
	return datatypes.WordClass{word1, likely}, nil
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
