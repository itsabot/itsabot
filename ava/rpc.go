package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

type Ava int

var regPkgs map[string]*pkg.Pkg
var client *rpc.Client

func bootRPCServer() {
	ava := new(Ava)
	rpc.Register(ava)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":4001")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// RegisterPackage enables Ava to notify packages when specific StructuredInput
// is encountered. Note that packages will only listen when ALL criteria are met
func (t *Ava) RegisterPackage(r *pkg.Pkg, reply *error) error {
	for _, c := range r.Trigger.Command {
		for _, o := range r.Trigger.Objects {
			for _, a := range r.Trigger.Actors {
				s := c + o + a
				if regPkgs[s] != nil {
					return fmt.Errorf(
						"duplicate package: %s",
						r.Config.Name)
				}
				regPkgs[s] = r
			}
		}
	}
	return nil
}

// callPkg finds the intersection of triggers
func callPkg(si *datatypes.StructuredInput) (string, error) {
	var p *pkg.Pkg
Loop:
	for _, c := range si.Command {
		for _, o := range si.Objects {
			for _, a := range si.Actors {
				p = regPkgs[c+o+a]
				if p != nil {
					break Loop
				}
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

// TODO: Setup client when package is registered, not when it's found
func foundPkg(p *pkg.Pkg, si *datatypes.StructuredInput) (string, error) {
	var err error
	client, err = rpc.DialHTTP("tcp", p.Config.ServerAddress+p.Config.Port)
	if err != nil {
		return "", err
	}
	log.Println("Sending to ", p.Config.Name)
	var reply string
	c := p.Config.Name + ":Run"
	err = client.Call(c, si, &reply)
	if err != nil {
		return "", err
	}
	return reply, nil
}
