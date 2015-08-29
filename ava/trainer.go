package main

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// delabelTrainingData generates StructuredInputs from training files for the
// purpose of testing and machine learning.
func delabelTrainingData(fp string) ([]StructuredInput, error) {
	var data []StructuredInput

	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		if len(t) >= 2 && t[0:2] != "//" {
			data = append(data, delabelSentence(t))
		}
	}
	return data, nil
}

// delabelSentence removes the sentence labels and builds a StructuredInput
// struct. This is used among training data. Note that compound sentences, e.g.
// "Order an Uber, and build a bridge," are not supported in this function.
func delabelSentence(s string) StructuredInput {
	var si StructuredInput

	rcmd := regexp.MustCompile(`_C\([\w\s-_/]+\)`)
	robj := regexp.MustCompile(`_O\([\w\s-_/]+\)`)
	ract := regexp.MustCompile(`_A\([\w\s-_/]+\)`)
	cmd := rcmd.FindString(s)
	obj := robj.FindAllString(s, -1)
	act := ract.FindAllString(s, -1)
	sr := strings.NewReplacer(
		"_C(", "",
		"_A(", "",
		"_O(", "",
		")", "")
	si.Sentence = sr.Replace(s)
	if len(cmd) > 3 {
		si.Command = cmd[3 : len(cmd)-1]
	}
	for _, o := range obj {
		if len(o) > 3 {
			si.Objects = append(si.Objects, o[3:len(o)-1])
		}
	}
	for _, a := range act {
		if len(a) > 3 {
			si.Actors = append(si.Actors, a[3:len(a)-1])
		}
	}
	return si
}
