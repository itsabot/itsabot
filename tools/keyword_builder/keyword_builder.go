package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/avabot/ava/shared/search"
)

type keyword struct {
	Name  string
	Count uint
	POS   string
}

func main() {
	client := search.NewClient()
	indexExists, err := client.IndicesExists("keywords")
	if err != nil {
		log.Fatalln("err", "checking exists", err)
		return
	}
	if !indexExists {
		_, err := client.CreateIndex("keywords")
		if err != nil {
			log.Fatalln("err", "elastic index", err)
			return
		}
	}
	buckets, err := client.FindProductKeywords("alcohol")
	if err != nil {
		log.Fatalln("err", "find product keywords", err)
	}
	// If yes, save to elasticsearch. Show a count of how many remaining.
	fmt.Println("Classify each keyword.")
	fmt.Println("N - Noun")
	fmt.Println("A - Adjective")
	fmt.Println("ADV - Adverb")
	fmt.Println("<blank> - None")
	reader := bufio.NewReader(os.Stdin)
	l := len(buckets)
	for i, b := range buckets {
		kw := b.Key
		fmt.Printf("%s (%d/%d): ", kw, i+1, l)
		var text string
	HandleInput:
		text, err = reader.ReadString('\n')
		if err != nil {
			log.Fatalln("err", err)
		}
		text = strings.ToLower(text)
		if text == "n\n" {
			// Noun
			keyw := keyword{Name: kw, Count: b.DocCount, POS: "n"}
			_, err = client.Index("keywords", "products_alcohol",
				kw, nil, keyw)
			if err != nil {
				log.Println("err", err)
				goto HandleInput
			}
		} else if text == "a\n" {
			// Adjective
			keyw := keyword{Name: kw, Count: b.DocCount, POS: "adj"}
			_, err = client.Index("keywords", "products_alcohol",
				kw, nil, keyw)
			if err != nil {
				log.Println("err", err)
				goto HandleInput
			}
		} else if text == "adv\n" {
			// Adverb
			keyw := keyword{Name: kw, Count: b.DocCount, POS: "adv"}
			_, err = client.Index("keywords", "products_alcohol",
				kw, nil, keyw)
			if err != nil {
				log.Println("err", err)
				goto HandleInput
			}
		} else if text == "\n" {
			_, err = client.Delete("keywords", "products_alcohol",
				kw, nil)
			if err != nil && err.Error() == "record not found" {
				continue
			} else if err != nil {
				log.Println("err", err)
				goto HandleInput
			}
		} else {
			fmt.Printf("Unrecognized option: %s", text)
			goto HandleInput
		}
	}
	fmt.Println("Done!")
}
