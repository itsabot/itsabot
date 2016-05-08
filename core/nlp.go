package core

import (
	"bufio"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/nlp"
)

// Classifier is a set of common english word stems unique among their
// Structured Input Types. This enables extremely fast constant-time O(1)
// lookups of stems to their SITs with high accuracy and no training
// requirements. It consumes just a few MB in memory.
type Classifier map[string]struct{}

// ClassifyTokens builds a StructuredInput from a tokenized sentence.
func (c Classifier) ClassifyTokens(tokens []string) *nlp.StructuredInput {
	var s nlp.StructuredInput
	for _, t := range tokens {
		t = strings.ToLower(t)
		_, exists := c["C"+t]
		if exists {
			s.Commands = append(s.Commands, t)
		}
		_, exists = c["O"+t]
		if exists {
			s.Objects = append(s.Objects, t)
		}
	}
	return &s
}

// buildClassifier prepares the Named Entity Recognizer (NER) to find Commands
// and Objects using a simple dictionary lookup. This has the benefit of high
// speed--constant time, O(1)--with insignificant memory use and high accuracy
// given false positives (marking something as both a Command and an Object when
// it's really acting as an Object) are OK. Utlimately this should be a first
// pass, and any double-marked words should be passed through something like an
// n-gram Bayesian filter to determine the correct part of speech within its
// context in the sentence.
func buildClassifier() (Classifier, error) {
	ner := Classifier{}
	p := filepath.Join("data", "ner")
	fi, err := os.Open(filepath.Join(p, "nouns.txt"))
	if err != nil {
		return ner, err
	}
	scanner := bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["O"+scanner.Text()] = struct{}{}
	}
	if err = fi.Close(); err != nil {
		return ner, err
	}
	fi, err = os.Open(filepath.Join(p, "verbs.txt"))
	if err != nil {
		return ner, err
	}
	scanner = bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["C"+scanner.Text()] = struct{}{}
	}
	if err = fi.Close(); err != nil {
		return ner, err
	}
	fi, err = os.Open(filepath.Join(p, "adjectives.txt"))
	if err != nil {
		return ner, err
	}
	scanner = bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["O"+scanner.Text()] = struct{}{}
	}
	if err = fi.Close(); err != nil {
		return ner, err
	}
	fi, err = os.Open(filepath.Join(p, "adverbs.txt"))
	if err != nil {
		return ner, err
	}
	scanner = bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["O"+scanner.Text()] = struct{}{}
	}
	if err = fi.Close(); err != nil {
		return ner, err
	}
	return ner, nil
}

// buildOffensiveMap creates a map of offensive terms for which Abot will refuse
// to respond. This helps ensure that users are somewhat respectful to Abot and
// her human trainers, since sentences caught by the OffensiveMap are rejected
// before any human ever sees them.
func buildOffensiveMap() (map[string]struct{}, error) {
	o := map[string]struct{}{}
	p := filepath.Join("data", "offensive.txt")
	fi, err := os.Open(p)
	if err != nil {
		return o, err
	}
	scanner := bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		o[scanner.Text()] = struct{}{}
	}
	err = fi.Close()
	return o, err
}

// RespondWithNicety replies to niceties that humans use, but Abot can ignore.
// Words like "Thank you" are not necessary for a robot, but it's important Abot
// respond correctly nonetheless.
func RespondWithNicety(in *dt.Msg) string {
	for _, w := range in.Stems {
		// Since these are stems, some of them look incorrectly spelled.
		// Needless to say, these are the correct Porter2 Snowball stems
		switch w {
		case "thank":
			return "You're welcome!"
		case "cool", "sweet", "awesom", "neat", "perfect":
			return "I know!"
		case "sorri":
			return "That's OK. I forgive you."
		case "hi", "hello":
			return "Hi there. :)"
		}
	}
	return ""
}

// RespondWithOffense is a one-off function to respond to rude user language by
// refusing to process the command.
func RespondWithOffense(off map[string]struct{}, in *dt.Msg) string {
	for _, w := range in.Stems {
		_, offensive := off[w]
		if offensive {
			return "I'm sorry, but I don't respond to rude language."
		}
	}
	return ""
}

// ConfusedLang returns a randomized response signalling that Abot is confused
// or could not understand the user's request.
func ConfusedLang() string {
	n := rand.Intn(4)
	switch n {
	case 0:
		return "I'm not sure I understand you."
	case 1:
		return "I'm sorry, I don't understand that."
	case 2:
		return "Uh, what are you telling me to do?"
	case 3:
		return "What should I do?"
	}
	log.Debug("confused failed to return a response")
	return ""
}
