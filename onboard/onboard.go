package main

import (
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

type Onboard string

func main() {
	plog = log.WithField("package", "onboard")
	trigger := &datatypes.StructuredInput{
		Command: []string{
			"hi",
			"hello",
			"greetings",
		},
	}
	p, err := pkg.NewPackage("onboard", trigger)
	if err != nil {
		plog.Fatal("creating package", p.Config.Name, err)
	}
	onboard := new(Onboard)
	if err := p.Register(onboard); err != nil {
		plog.Fatal("registering package ", err)
	}
}

func (t *Onboard) Run(si *datatypes.StructuredInput, resp *string) error {
	plog.Debug("package called")
	return nil
}
