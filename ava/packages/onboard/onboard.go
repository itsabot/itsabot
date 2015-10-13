package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/NickPresta/GoURLShortener"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

var port = flag.Int("port", 0, "Port used to communicate with Ava.")

type Onboard string

func main() {
	flag.Parse()
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
	u, err := getURL(m)
	if err != nil {
		return err
	}
	*resp = "Hi, I'm Ava. To get started, you can sign up here: " + u
	return nil
}

func (t *Onboard) FollowUp(m *datatypes.Message, resp *string) error {
	u, err := getURL(m)
	if err != nil {
		return err
	}
	*resp = "Hi, I'm Ava. To get started, you can sign up here: " + u
	return nil
}

func getURL(m *datatypes.Message) (string, error) {
	fid := m.Input.FlexId
	fidT := m.Input.FlexIdType
	v := url.Values{
		"flexid":     {fid},
		"flexidtype": {strconv.Itoa(fidT)},
	}
	v.Set("encodedpath", v.Encode())
	u := os.Getenv("BASE_URL") + "signup?" + v.Encode()
	u, err := goisgd.Shorten(u)
	if err != nil {
		return "", err
	}
	return u, nil
}
