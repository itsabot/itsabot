package main

import (
	"bufio"
	"os"

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
