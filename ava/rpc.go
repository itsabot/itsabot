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
	pt := p.Config.Port + 1
	log.WithField("port", pt).Debug("registering package with listen port")
	port := ":" + strconv.Itoa(pt)
	addr := p.Config.ServerAddress + port
	cl, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}
	for _, c := range p.Trigger.Commands {
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + o)
			if regPkgs[s] != nil {
				return fmt.Errorf(
					"duplicate package or trigger: %s",
					p.Config.Name)
			}
			logger := log.WithField("package", p.Config.Name)
			regPkgs[s] = &pkg.PkgWrapper{
				P:         p,
				RPCClient: cl,
				Logger:    logger,
			}
		}
	}
	return nil
}

func getPkg(si *datatypes.StructuredInput) (*pkg.PkgWrapper, string, error) {
	var p *pkg.PkgWrapper
	var route string
Loop:
	for _, c := range si.Commands {
		for _, o := range si.Objects {
			route = strings.ToLower(c + o)
			p = regPkgs[route]
			log.Debug("searching for " + strings.ToLower(c+o))
			if p != nil {
				log.Debug("found pkg")
				break Loop
			}
		}
	}
	if p == nil {
		return nil, "", ErrMissingPackage
	} else {
		return p, route, nil
	}
}

func callPkg(pw *pkg.PkgWrapper, si *datatypes.StructuredInput, ctxAdded bool) (
	string, error) {
	log.Debug("sending structured input to ", pw.P.Config.Name)
	c := strings.Title(pw.P.Config.Name)
	if ctxAdded {
		c += ".FollowUp"
	} else {
		c += ".Run"
	}
	var reply string
	if err := pw.RPCClient.Call(c, si, &reply); err != nil {
		return "", err
	}
	log.Debug("r: ", reply)
	return reply, nil
}
