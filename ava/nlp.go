package main

import (
	"log"
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

func buildStructuredInput(nl string) StructuredInput {
	var si StructuredInput

	sentences := strings.Split(nl, ".")
	for _, sent := range sentences {
		si = si.add(parseSentence(sent))
	}
	return si
}

func parseSentence(s string) StructuredInput {
	var buf string
	var fndType int

	si := StructuredInput{Sentence: s}
	words := strings.Fields(s)
	for _, word := range words {
		word = strings.TrimRight(word, ",.")
		key := strings.ToLower(word)
		si, fndType = si.addIfFound(key, word, &buf)
		log.Println("found", fndType)
		switch fndType {
		case 0:
			log.Println("buf", buf)
		case nlVerb, nlPreposition, nlName:
			buf = ""
		case nlNoun:
			log.Println("modifying obj", si.Objects[len(si.Objects)-1])
			buf = ""
		}
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

func (s StructuredInput) addIfFound(k, w string, buf *string) (StructuredInput,
	int) {
	//engine := freeling.NewEngine()
	//engine.InitNLP()
	/*
		found := dict[k]
		if found == 0 {
			//found = dict[inflect.Singularize(k)]
		}
		switch found {
		case nlVerb:
			if s.Command != "" {
				log.Println("warning: overriding command", s.Command)
			}
			s.Command = w
		case nlNoun:
			if len(*buf) > 0 {
				obj := s.Objects[len(s.Objects)-1]
				s.Objects[len(s.Objects)-1] = *buf + " " + obj
			}
			s.Objects = append(s.Objects, w)
		case nlName:
			if len(*buf) > 0 {
				obj := s.Actors[len(s.Actors)-1]
				s.Actors[len(s.Actors)-1] = *buf + " " + obj
			}
			s.Actors = append(s.Actors, w)
		case 0:
			if len(*buf) == 0 {
				*buf = w
			} else {
				*buf += " " + w
			}
		}
		return s, int(found)
	*/
	return s, 0
}

// TODO
// breakCompoundSent splits compound sentences into different sentences for
// ease of parsing.
func breakCompoundSent(sentences []string) [][]string {
	return [][]string{}
}

/*
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
*/
