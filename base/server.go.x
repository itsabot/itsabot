package main

import (
	"net/http"
	"os"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
)

func main() {
	hr, err := core.NewServer()
	if err != nil {
		log.Fatalf("could not start server. %s", err)
	}
	log.Info("started abot")
	if err = http.ListenAndServe(":"+os.Getenv("PORT"), hr); err != nil {
		log.Fatalf("could not listen on port. %s", err)
	}
}
