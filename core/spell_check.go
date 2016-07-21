package core

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/itsabot/abot/core/log"
	"github.com/sajari/fuzzy"
)

var spellChecker *fuzzy.Model

// trainSpellCheck model on a corpus.
func trainSpellCheck() error {
	log.Info("training spell checker")
	spellChecker = fuzzy.NewModel()
	spellChecker.SetDepth(2)
	spellChecker.SetThreshold(4)
	spellChecker.SetUseAutocomplete(false)
	var p, p2 string
	if os.Getenv("ABOT_ENV") == "test" {
		p = filepath.Join(os.Getenv("ABOT_PATH"), "base", "data",
			"coca.txt")
		p2 = filepath.Join(os.Getenv("ABOT_PATH"), "base", "data",
			"american-english.txt")
	} else {
		p = filepath.Join("data", "coca.txt")
		p2 = filepath.Join("data", "american-english.txt")
	}
	if err := trainFile(p, 1); err != nil {
		return err
	}
	if err := trainFile(p2, 3); err != nil {
		return err
	}
	return nil
}

// spellCheckTokens corrects the spelling of tokens received.
func spellCheckTokens(tokens []string) []string {
	c := make(chan struct {
		Token string
		Idx   int
	}, len(tokens))
	ts := make([]string, len(tokens))
	wg := &sync.WaitGroup{}
	wg.Add(len(tokens))
	go func() {
		for {
			select {
			case token := <-c:
				ts[token.Idx] = token.Token
				wg.Done()
			}
		}
	}()
	for i, token := range tokens {
		go func(i int, token string) {
			tmp := spellChecker.SpellCheck(token)
			if len(tmp) > 0 {
				token = tmp
			}
			c <- struct {
				Token string
				Idx   int
			}{Token: token, Idx: i}
		}(i, token)
	}
	wg.Wait()
	return ts
}

func trainFile(p string, weight int) error {
	fi, err := os.Open(p)
	if err != nil {
		return err
	}
	defer func() {
		if err = fi.Close(); err != nil {
			log.Info("failed to close COCA corpus file.", err)
		}
	}()
	scn := bufio.NewScanner(fi)
	exp := regexp.MustCompile("[a-zA-Z]+")
	for scn.Scan() {
		words := exp.FindAllString(scn.Text(), -1)
		for _, word := range words {
			for i := 0; i < weight; i++ {
				spellChecker.TrainWord(word)
			}
		}
	}
	if scn.Err() != nil {
		return scn.Err()
	}
	return nil
}
