package main

import (
	"errors"
	"flag"
	"net/url"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
)

var port = flag.Int("port", 0, "Port used to communicate with Ava.")

var plog *log.Entry

type Onboard string

func main() {
	if os.Getenv("AVA_ENV") == "production" {
		log.SetLevel(log.WarnLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
	plog = log.WithField("package", "onboard")
	flag.Parse()
	// NOTE ONBOARD is a special trigger that's called when a user cannot be
	// found for a particular flexid. Normally triggers must be lowercase.
	trigger := &datatypes.StructuredInput{
		Commands: []string{"ONBOARD"},
	}
	p, err := pkg.NewPackage("onboard", *port, trigger)
	if err != nil {
		plog.Fatal("creating package", p.Config.Name, err)
	}
	onboard := new(Onboard)
	if err := p.Register(onboard); err != nil {
		plog.Fatal("registering package ", err)
	}
}

func (t *Onboard) Run(m *datatypes.Message, resp *string) error {
	*resp = language.FirstMeeting()
	return nil
}

func (t *Onboard) FollowUp(m *datatypes.Message, resp *string) error {
	base := os.Getenv("BASE_URL")
	l := len(base)
	if l == 0 {
		return errors.New("BASE_URL environment variable not set")
	}
	if l < 4 || base[0:4] != "http" {
		return errors.New("BASE_URL invalid. Must include http/https")
	}
	if base[l-1] != '/' {
		base += "/"
	}
	name := url.QueryEscape(m.Input.StructuredInput.Objects.String())
	url := base + "login?name=" + name
	*resp = language.NiceMeetingYou() + url
	return nil
}
