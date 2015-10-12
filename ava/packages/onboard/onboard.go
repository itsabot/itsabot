package main

import (
	"flag"
	"log"
	"os"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

var port = flag.Int("port", 0, "Port used to communicate with Ava.")

type Onboard string

func main() {
	flag.Parse()
	// NOTE onboard is a special trigger that's called when a user cannot be
	// found for a particular flexid. Normally triggers must be lowercase.
	trigger := &datatypes.StructuredInput{
		Commands: []string{"onboard"},
	}
	p, err := pkg.NewPackage("onboard", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	onboard := new(Onboard)
	if err := p.Register(onboard); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (t *Onboard) Run(m *datatypes.Message, resp *string) error {
	url := os.Getenv("BASE_URL") + "signup"
	*resp = "To get started, sign up here: " + url
	return nil
}

func (t *Onboard) FollowUp(m *datatypes.Message, resp *string) error {
	url := os.Getenv("BASE_URL") + "login"
	*resp = "Please signup to get started: " + url
	return nil
}
