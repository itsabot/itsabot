package main

import (
	"log"
	"path"
	"reflect"
	"testing"
)

var test *testing.T

func TestBuildTrainingData(t *testing.T) {
	var err error

	si, err := delabelTrainingData(path.Join("training", "imperative.txt"))
	if err != nil {
		log.Println("failed delabeling training data", err)
		t.Fail()
	}
	for _, s := range si {
		p := parseSentence(s.Sentence)
		log.Println("PARSED", p)
		if !reflect.DeepEqual(s, p) {
			log.Println("failed delabeling sentence: ", s.Sentence)
			t.Fail()
		}
	}
}
