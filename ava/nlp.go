package main

import (
	"io/ioutil"
	"log"
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

const (
	nlNoun = iota + 1
	nlVerb
	nlPreposition
	nlName
)

var wordType map[string]int8 = map[string]int8{
	"nouns.txt":        nlNoun,
	"verbs.txt":        nlVerb,
	"prepositions.txt": nlPreposition,
	"names.txt":        nlName,
}

func loadDictionary() (wMap, error) {
	dict := wMap{}
	baseDir := path.Join("data", "lang-en")
	files, err := ioutil.ReadDir(baseDir)
	if err != nil {
		return dict, err
	}
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

func buildStructuredInput(nl string) StructuredInput {
	var si StructuredInput

	sentences := strings.Split(nl, ".")
	for _, sent := range sentences {
		si = si.add(parseSentence(sent))
	}
	return si
}

func parseSentence(s string) StructuredInput {
	si := StructuredInput{Sentence: s}
	words := strings.Fields(s)
	for _, word := range words {
		key := strings.ToLower(word)
		si = si.addIfFound(key, word)
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

func (s StructuredInput) addIfFound(key, word string) StructuredInput {
	w := dict[key]
	switch w {
	case nlVerb:
		if s.Command != "" {
			log.Println("warning: overriding command", s.Command)
		}
		s.Command = word
	case nlNoun:
		s.Objects = append(s.Objects, word)
	case nlName:
		s.Actors = append(s.Actors, word)
	}
	return s
}

// TODO
// breakCompoundSent splits compound sentences into different sentences for
// ease of parsing.
func breakCompoundSent(sentences []string) [][]string {
	return [][]string{}
}
