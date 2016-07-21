package main

import (
	"bufio"
	"log"
	"os"

	"github.com/dchest/stemmer/porter2"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer func() {
		if err := out.Flush(); err != nil {
			log.Fatal(err)
		}
	}()

	eng := porter2.Stemmer
	for word, err := in.ReadSlice('\n'); err == nil; word, err = in.ReadSlice('\n') {
		_, err := out.Write([]byte(eng.Stem(string(word))))
		if err != nil {
			log.Fatal(err)
		}
		_, err = out.WriteString("\n")
		if err != nil {
			log.Fatal(err)
		}
	}
}
