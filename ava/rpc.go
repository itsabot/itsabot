package main

import (
	"fmt"
	"net/rpc"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

type Ava int

var regPkgs map[string]*pkg.PkgWrapper = map[string]*pkg.PkgWrapper{}
var client *rpc.Client

// RegisterPackage enables Ava to notify packages when specific StructuredInput
// is encountered. Note that packages will only listen when ALL criteria are met
func (t *Ava) RegisterPackage(p *pkg.Pkg, reply *error) error {
	plog := log.WithField("package", p.Config.Name)
	plog.Debug("registering package")
	for _, c := range p.Trigger.Command {
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + o)
			if regPkgs[s] != nil {
				return fmt.Errorf(
					"duplicate package: %s",
					p.Config.Name)
			}
			var err error
			port := ":" + strconv.Itoa(p.Config.Port)
			addr := p.Config.ServerAddress + port
			client, err = rpc.Dial("tcp", addr)
			if err != nil {
				return err
			}
			logger := log.WithField("package", p.Config.Name)
			regPkgs[s] = &pkg.PkgWrapper{
				P:         p,
				RPCClient: client,
				Logger:    logger,
			}
		}
	}
	return nil
}

// callPkg finds the intersection of triggers
func callPkg(uid string, si *datatypes.StructuredInput) (string, error) {
	var p *pkg.PkgWrapper
Loop:
	for _, c := range si.Command {
		for _, o := range si.Objects {
			p = regPkgs[strings.ToLower(c+o)]
			if p != nil {
				break Loop
			}
		}
	}
	if p == nil {
		return "", ErrMissingPackage
	} else {
		resp, err := foundPkg(p, si)
		if err != nil {
			return "", err
		}
		return resp, nil
	}
}

func foundPkg(pw *pkg.PkgWrapper, si *datatypes.StructuredInput) (string, error) {
	log.Debug("sending structured input to ", pw.P.Config.Name)
	c := pw.P.Config.Name + ".Run"
	var reply string
	if err := pw.RPCClient.Call(c, si, &reply); err != nil {
		return "", err
	}
	return reply, nil
}
