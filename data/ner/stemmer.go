package main

import (
	"bufio"
	"log"
	"os"

	"github.com/agonopol/go-stem"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer func() {
		if err := out.Flush(); err != nil {
			log.Fatal(err)
		}
	}()

	for word, err := in.ReadSlice('\n'); err == nil; word, err = in.ReadSlice('\n') {
		_, err := out.Write(stemmer.Stem(word))
		if err != nil {
			log.Fatal(err)
		}
		_, err = out.WriteString("\n")
		if err != nil {
			log.Fatal(err)
		}
	}
}
