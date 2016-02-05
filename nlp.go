package main

import (
	"bufio"
	"os"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
)

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
// Words like "Thank you" are not necessary with a robot, but it's important Ava
// respond correctly. The returned bool specifies whether a response is
// necessary, and the returned string is the response, if any.
func respondWithNicety(in *dt.Msg) (bool, string) {
	for _, w := range in.Stems {
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

func respondWithOffense(off map[string]struct{}, in *dt.Msg) string {
	for _, w := range in.Stems {
		_, offensive := off[w]
		if offensive {
			return "I'm sorry, but I don't respond to rude language."
		}
	}
	return ""
}
