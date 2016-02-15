package main

import (
	"bufio"
	"os"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
)

// buildClassifier prepares the Named Entity Recognizer (NER) to find Commands
// and Objects using a simple dictionary lookup. This has the benefit of high
// speed--constant time, O(1)--with insignificant memory use and high accuracy
// given false positives (marking something as both a Command and an Object when
// it's really acting as an Object) are OK. Utlimately this should be a first
// pass, and any double-marked words should be passed through something like an
// n-gram Bayesian filter to determine the correct part of speech within its
// context in the sentence.
func buildClassifier() (nlp.Classifier, error) {
	ner := nlp.Classifier{}
	fi, err := os.Open("data/ner/nouns.txt")
	if err != nil {
		return ner, err
	}
	defer fi.Close()
	scanner := bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["O"+scanner.Text()] = struct{}{}
	}
	fi2, err := os.Open("data/ner/verbs.txt")
	if err != nil {
		return ner, err
	}
	defer fi2.Close()
	scanner = bufio.NewScanner(fi2)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["C"+scanner.Text()] = struct{}{}
	}
	fi3, err := os.Open("data/ner/adjectives.txt")
	if err != nil {
		return ner, err
	}
	defer fi3.Close()
	scanner = bufio.NewScanner(fi3)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["O"+scanner.Text()] = struct{}{}
	}
	fi4, err := os.Open("data/ner/adverbs.txt")
	if err != nil {
		return ner, err
	}
	defer fi4.Close()
	scanner = bufio.NewScanner(fi4)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		ner["O"+scanner.Text()] = struct{}{}
	}
	return ner, nil
}

// buildOffensiveMap creates a map of offensive terms for which Ava will refuse
// to respond. This helps ensure that users are somewhat respectful to Ava and
// her human trainers, since sentences caught by the OffensiveMap are rejected
// before any human ever sees them.
func buildOffensiveMap() (map[string]struct{}, error) {
	o := map[string]struct{}{}
	fi, err := os.Open("data/offensive.txt")
	if err != nil {
		return o, err
	}
	defer fi.Close()
	scanner := bufio.NewScanner(fi)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		o[scanner.Text()] = struct{}{}
	}
	return o, nil
}

// respondWithNicety replies to niceties that humans use, but Ava can ignore.
// Words like "Thank you" are not necessary for a robot, but it's important Ava
// respond correctly nonetheless. The returned bool specifies whether a response
// is necessary, and the returned string is the response, if any.
func respondWithNicety(in *dt.Msg) (responseNecessary bool, response string) {
	for _, w := range in.Stems {
		// Since these are stems, some of them look incorrectly spelled.
		// Needless to say, these are the correct Porter2 Snowball stems
		switch w {
		case "thank":
			return true, "You're welcome!"
		case "cool", "sweet", "awesom", "neat", "perfect":
			return false, ""
		case "sorri":
			return true, "That's OK. I forgive you."
		}
	}
	return true, ""
}

// respondWithOffense is a one-off function to respond to rude user language by
// refusing to process the command.
func respondWithOffense(off map[string]struct{}, in *dt.Msg) string {
	for _, w := range in.Stems {
		_, offensive := off[w]
		if offensive {
			return "I'm sorry, but I don't respond to rude language."
		}
	}
	return ""
}
