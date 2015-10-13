package main

import (
	"flag"
	"fmt"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/NickPresta/GoURLShortener"
	"os"
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Cowardly refusing to shorten a blank URL")
		return
	}

	uri, err := goisgd.Shorten(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	fmt.Println(uri)
	return
}
