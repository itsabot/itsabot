package main

import (
	"fmt"
	"log"
	"net/rpc"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

type Ava int

type pkgMap struct {
	pkgs  map[string]*pkg.PkgWrapper
	mutex *sync.Mutex
}

var regPkgs = pkgMap{
	pkgs:  make(map[string]*pkg.PkgWrapper),
	mutex: &sync.Mutex{},
}

var client *rpc.Client

// RegisterPackage enables Ava to notify packages when specific StructuredInput
// is encountered. Note that packages will only listen when ALL criteria are met
func (t *Ava) RegisterPackage(p *pkg.Pkg, reply *string) error {
	pt := p.Config.Port + 1
	log.Println("registering package with listen port", pt)
	port := ":" + strconv.Itoa(pt)
	addr := p.Config.ServerAddress + port
	cl, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}
	for _, c := range p.Trigger.Commands {
		c = strings.ToLower(c)
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + "_" + o)
			if regPkgs.Get(s) != nil {
				log.Println(
					"warn: duplicate package or trigger",
					p.Config.Name, s)
			}
			regPkgs.Set(s, &pkg.PkgWrapper{P: p, RPCClient: cl})
		}
		regPkgs.Set(c, &pkg.PkgWrapper{P: p, RPCClient: cl})
	}
	return nil
}

func getPkg(m *dt.Msg) (*pkg.PkgWrapper, string, bool, error) {
	var p *pkg.PkgWrapper
	if m.User == nil {
		p = regPkgs.Get("onboard")
		if p != nil {
			return p, "onboard", false, nil
		} else {
			log.Println("err: missing required onboard package")
			return nil, "onboard", false, ErrMissingPackage
		}
	}
	var route string
	var shortRoute string
	si := m.Input.StructuredInput
Loop:
	for _, c := range si.Commands {
		c = strings.Split(c, "'")[0]
		for _, o := range si.Objects {
			o = strings.Split(o, "'")[0]
			route = strings.ToLower(c + "_" + o)
			p = regPkgs.Get(route)
			log.Println("searching for " + strings.ToLower(c+"_"+o))
			if p != nil {
				shortRoute = ""
				break Loop
			}
			p = regPkgs.Get(o)
			if p != nil {
				shortRoute = o
			}
		}
		p = regPkgs.Get(c)
		if p != nil {
			shortRoute = c
		}
	}
	if len(shortRoute) > 0 {
		route = shortRoute
	}
	if p == nil {
		log.Println("p is nil, getting last response route")
		if err := m.GetLastResponse(db); err != nil {
			return p, route, false, err
		}
		if m.LastResponse == nil {
			log.Println("couldn't find last package")
			return p, route, false, ErrMissingPackage
		}
		route = m.LastResponse.Route
		p = regPkgs.Get(route)
		if p == nil {
			log.Println("HERE")
			return p, route, true, ErrMissingPackage
		}
		// TODO pass LastResponse directly to packages via rpc gob
		// encoding, removing the need to nil this out and then look it
		// up again in the package
		m.LastResponse = nil
		return p, route, false, nil
	} else {
		return p, route, false, nil
	}
}

func callPkg(m *dt.Msg, ctxAdded bool) (*dt.RespMsg, string, string, error) {
	reply := &dt.RespMsg{}
	pw, route, lastRoute, err := getPkg(m)
	if err != nil {
		var pname string
		if pw != nil {
			pname = pw.P.Config.Name
		}
		return reply, pname, route, err
	}
	log.Println("sending structured input to", pw.P.Config.Name)
	c := strings.Title(pw.P.Config.Name)
	if ctxAdded || lastRoute || len(m.Input.StructuredInput.Commands) == 0 {
		log.Println("FollowUp")
		c += ".FollowUp"
	} else {
		c += ".Run"
	}
	m.Route = route
	log.Println("calling pkg with", fmt.Sprintf("%+v", m))
	if err := pw.RPCClient.Call(c, m, reply); err != nil {
		log.Println("invalid response")
		return reply, pw.P.Config.Name, route, err
	}
	return reply, pw.P.Config.Name, route, nil
}

func (pm pkgMap) Get(k string) *pkg.PkgWrapper {
	var pw *pkg.PkgWrapper
	pm.mutex.Lock()
	pw = pm.pkgs[k]
	pm.mutex.Unlock()
	runtime.Gosched()
	return pw
}

func (pm pkgMap) Set(k string, v *pkg.PkgWrapper) {
	pm.mutex.Lock()
	pm.pkgs[k] = v
	pm.mutex.Unlock()
	runtime.Gosched()
}
