package main

import (
	"bufio"
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/agonopol/go-stem"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for word, err := in.ReadSlice('\n'); err == nil; word, err = in.ReadSlice('\n') {
		out.Write(stemmer.Stem(word))
		out.WriteString("\n")
	}
}
