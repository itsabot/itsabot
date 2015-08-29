package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

type StructuredInput struct {
	Sentence string
	Command  string
	Actors   []string
	Objects  []string
}

type wMap map[string]int8

var wordType map[string]int8 = map[string]int8{
	"nouns.txt":        nlNoun,
	"verbs.txt":        nlVerb,
	"prepositions.txt": nlPreposition,
	"names.txt":        nlName,
}

const (
	nlNoun = iota + 1
	nlVerb
	nlPreposition
	nlName
	nlArticle
)

func loadDictionary() (wMap, error) {
	baseDir := path.Join("data", "lang-en")
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return c, err
	}
	var dict wMap
	for _, file := range files {
		content, err := ioutil.ReadFile(path.Join(baseDir, file.Name()))
		if err != nil {
			return dict, err
		}
		words := strings.Split(string(content), "\n")
		for _, word := range words {
			dict[word] = wordType[file.Name()]
		}
	}
	return dict, nil
}

func labelTrainingData(fp string) error {
	var data []string

	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		if t[0:1] != "//" || t[0] == "\n" {
			data = append(data, delabelSentence(t))
		}
	}
	return nil
}

// TODO: HERE
func delabelSentence(s string) StructuredInput {
	var ss StructuredInput
	var labelF, wordF bool

	for _, l := range s {
		switch l {
		case '_':
			labelF = true
		case '(':
			continue
		case ')':
			wordF = false
		case 'C':
			if labelF {
				labelF = false
			}
		default:
			if labelF {
				ss.Command
			}
			if wordF {
				word = append(word, l)
			} else if labelF {

			}
		}
	}
}

func buildStructuredInput(nl string) StructuredInput {
	var si StructuredInput
	sentences := strings.Split(nl, ".")
	for _, sent := range sentences {
		si = si.add(parseSentence(sent))
	}
	return si
}

func parseSentence(s string) StructuredInput {
	var si StructuredInput
	s := strings.NewReplacer(
		"!", "",
		".", "",
		",", "",
		"\"", "",
		"'", "",
		"-", "",
	)
	words := strings.Fields(s)
	for i, word := range words {
		si = si.addIfFound(word, words, i)
	}
	return si
}

func (s StructuredInput) add(ns StructuredInput) StructuredInput {
	if s.Command == "" {
		s.Command = ns.Command
	}
	for _, person := range ns.Actors {
		s.Actors = append(s.Actors, person)
	}
	for _, object := range ns.Objects {
		s.Objects = append(s.Objects, object)
	}
	return s
}

func (s StructuredInput) addIfFound(word string, words []string, index int) StructuredInput {
	w := dict[word]
	switch w {
	case nlVerb:
		s.Command = word
	case nlNoun:
		s.Objects = append(s.Objects, word)
	case nlName:
		s.Actors = append(s.Actors, word)
	default:
		log.Println("word not found:", word)
	}
	return s
}

type verbContext struct {
	verb       string
	verbIndex  int
	directObjs []string
}

// TODO
// breakCompoundSent splits compound sentences into different sentences for
// ease of parsing.
func breakCompoundSent(sentence []string) [][]string {
	for _, s := range sentences {

	}
}
