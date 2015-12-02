package main

import (
	"flag"
	"log"
	"net/url"
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/NickPresta/GoURLShortener"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

var port = flag.Int("port", 0, "Port used to communicate with Ava.")

var p *pkg.Pkg

type Onboard string

func main() {
	flag.Parse()
	trigger := &dt.StructuredInput{
		Commands: []string{"onboard"},
		Objects:  []string{"onboard"},
	}
	var err error
	p, err = pkg.NewPackage("onboard", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	onboard := new(Onboard)
	if err := p.Register(onboard); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (t *Onboard) Run(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	u, err := getURL(m)
	if err != nil {
		return err
	}
	log.Println("M", m)
	resp := m.NewResponse()
	resp.Sentence = "Hi, I'm Ava, your new personal assistant. To get started, please sign up here: " + u
	return p.SaveResponse(respMsg, resp)
}

func (t *Onboard) FollowUp(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	u, err := getURL(m)
	if err != nil {
		return err
	}
	resp := m.NewResponse()
	resp.Sentence = "Hi, I'm Ava. To get started, you can sign up here: " + u
	return p.SaveResponse(respMsg, resp)
}

func getURL(m *dt.Msg) (string, error) {
	fid := m.Input.FlexID
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
