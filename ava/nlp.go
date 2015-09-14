package main

import (
	"container/list"
	"log"
	"strings"

	"github.com/egtann/freeling/models"
	fnlp "github.com/egtann/freeling/nlp"
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

	body := ""
	tokens := list.New()
	if nlp.Tokenizer != nil {
		nlp.Tokenizer.Tokenize(s, 0, tokens)
	}
	sentences := list.New()

	if nlp.Splitter != nil {
		sid := nlp.Splitter.OpenSession()
		nlp.Splitter.Split(sid, tokens, true, sentences)
		nlp.Splitter.CloseSession(sid)
	}

	for ss := sentences.Front(); ss != nil; ss = ss.Next() {
		s := ss.Value.(*fnlp.Sentence)
		if nlp.Morfo != nil {
			nlp.Morfo.Analyze(s)
		}
		if nlp.Sense != nil {
			nlp.Sense.Analyze(s)
		}
		if nlp.Tagger != nil {
			nlp.Tagger.Analyze(s)
		}
		if nlp.ShallowParser != nil {
			nlp.ShallowParser.Analyze(s)
		}
	}
	if nlp.Dsb != nil {
		nlp.Dsb.Analyze(sentences)
	}
	entities := make(map[string]int64)
	for ss := sentences.Front(); ss != nil; ss = ss.Next() {
		se := models.NewSentenceEntity()
		body := ""
		s := ss.Value.(*fnlp.Sentence)
		for ww := s.Front(); ww != nil; ww = ww.Next() {
			w := ww.Value.(*fnlp.Word)
			a := w.Front().Value.(*fnlp.Analysis)
			te := models.NewTokenEntity(w.getForm(), a.getLemma(), a.getTag(), a.getProb())
			if a.getTag() == "NP" {
				entities[w.getForm()]++
			}
			body += w.getForm() + " "
			se.AddTokenEntity(te)
		}
		body = strings.Trim(body, " ")
		se.SetBody(body)
		se.SetSentence(s)
	}
	log.Println("BODY", body)

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
	return s, 0
}

// TODO
// breakCompoundSent splits compound sentences into different sentences for
// ease of parsing.
func breakCompoundSent(sentences []string) [][]string {
	return [][]string{}
}
