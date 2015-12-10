package main

import (
	"net/rpc"
	"runtime"
	"strconv"
	"strings"
	"sync"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
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
	log.WithFields(log.Fields{
		"pkg":  p.Config.Name,
		"port": pt,
	}).Debugln("registering")
	port := ":" + strconv.Itoa(pt)
	addr := p.Config.ServerAddress + port
	cl, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}
	for _, c := range p.Trigger.Commands {
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + "_" + o)
			if regPkgs.Get(s) != nil {
				log.WithFields(log.Fields{
					"pkg":   p.Config.Name,
					"route": s,
				}).Warnln("duplicate package or trigger")
			}
			regPkgs.Set(s, &pkg.PkgWrapper{P: p, RPCClient: cl})
		}
	}
	return nil
}

func getPkg(m *dt.Msg) (*pkg.PkgWrapper, string, bool, error) {
	var p *pkg.PkgWrapper
	if m.User == nil {
		p = regPkgs.Get("onboard_onboard")
		if p != nil {
			return p, "onboard_onboard", false, nil
		} else {
			log.Errorln("missing required onboard package")
			return nil, "onboard_onboard", false, ErrMissingPackage
		}
	}
	var route string
	si := m.Input.StructuredInput
Loop:
	for _, c := range si.Commands {
		for _, o := range si.Objects {
			route = strings.ToLower(c + "_" + o)
			p = regPkgs.Get(route)
			if p != nil {
				break Loop
			}
		}
	}
	if p == nil {
		log.Infoln("p is nil, getting last response route")
		if err := m.GetLastResponse(db); err != nil {
			return p, route, false, err
		}
		if m.LastResponse == nil {
			log.Infoln("couldn't find last package")
			return p, route, false, ErrMissingPackage
		}
		route = m.LastResponse.Route
		p = regPkgs.Get(route)
		if p == nil {
			return p, route, true, ErrMissingPackage
		}
		// TODO pass LastResponse directly to packages via rpc gob
		// encoding, removing the need to nil this out and then look it
		// up again in the package
		m.LastResponse = nil
		return p, route, true, nil
	} else {
		return p, route, false, nil
	}
}

func callPkg(m *dt.Msg) (*dt.RespMsg, string, string, error) {
	reply := &dt.RespMsg{}
	pw, route, lastRoute, err := getPkg(m)
	if err != nil {
		log.WithField("fn", "callPkg:getPkg").Errorln(err)
		var pname string
		if pw != nil {
			pname = pw.P.Config.Name
		}
		return reply, pname, route, err
	}
	log.WithField("pkg", pw.P.Config.Name).Infoln("sending input")
	c := strings.Title(pw.P.Config.Name)
	if lastRoute || len(m.Input.StructuredInput.Commands) == 0 {
		log.WithField("pkg", pw.P.Config.Name).Infoln("follow up")
		c += ".FollowUp"
	} else {
		log.WithField("pkg", pw.P.Config.Name).Infoln("first run")
		c += ".Run"
	}
	m.Route = route
	if err := pw.RPCClient.Call(c, m, reply); err != nil {
		log.WithField("pkg", pw.P.Config.Name).Errorln(
			"invalid response", err)
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
