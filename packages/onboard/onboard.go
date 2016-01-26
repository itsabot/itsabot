package main

import (
	"flag"
	"net/url"
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/NickPresta/GoURLShortener"
	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
	"github.com/avabot/ava/shared/pkg"
)

var port = flag.Int("port", 0, "Port used to communicate with Ava.")

var p *pkg.Pkg
var l *log.Entry

type Onboard string

func main() {
	flag.Parse()
	l = log.WithFields(log.Fields{
		"pkg": "onboard",
	})
	trigger := &nlp.StructuredInput{
		Commands: []string{"onboard"},
		Objects:  []string{"onboard"},
	}
	var err error
	p, err = pkg.NewPackage("onboard", *port, trigger)
	if err != nil {
		l.Fatalln("building", err)
	}
	onboard := new(Onboard)
	if err := p.Register(onboard); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Onboard) Run(m *dt.Msg, resp *string) error {
	u, err := getURL(m)
	if err != nil {
		return err
	}
	*resp = "Hi, I'm Ava, your new personal assistant. To get started, please sign up here: " + u
	return nil
}

func (t *Onboard) FollowUp(m *dt.Msg, resp *string) error {
	u, err := getURL(m)
	if err != nil {
		return err
	}
	*resp = "Hi, I'm Ava. To get started, you can sign up here: " + u
	return nil
}

// TODO fix, passing in flexid to msg
func getURL(m *dt.Msg) (string, error) {
	fid := m.FlexID
	v := url.Values{
		"fid": {fid},
	}
	u := os.Getenv("BASE_URL") + "?/signup?" + v.Encode()
	u, err := goisgd.Shorten(u)
	if err != nil {
		return "", err
	}
	return u, nil
}
