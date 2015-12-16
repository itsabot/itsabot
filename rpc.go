package main

import (
	"database/sql"
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

type atomicMap struct {
	words map[string]bool
	mutex *sync.Mutex
}

var appVocab atomicMap

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
		appVocab.Set(c, true)
		for _, o := range p.Trigger.Objects {
			appVocab.Set(o, true)
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
	if p.Vocab != nil {
		for k := range p.Vocab.Commands {
			appVocab.Set(k, true)
		}
		for k := range p.Vocab.Objects {
			appVocab.Set(k, true)
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
	err := m.GetLastResponse(db)
	if err != nil && err != sql.ErrNoRows {
		return nil, "", false, err
	} else if err == sql.ErrNoRows {
		log.Errorln("no rows for last response")
	}
	var route string
	si := m.Input.StructuredInput
Loop:
	for _, c := range si.Commands {
		for _, o := range si.Objects {
			route = strings.ToLower(c + "_" + o)
			log.Debugln("searching route", route)
			p = regPkgs.Get(route)
			if p != nil {
				break Loop
			}
		}
	}
	if p != nil {
		return p, route, false, nil
	}
	if m.LastResponse == nil {
		log.Warnln("no last response")
		return p, route, false, nil
	}
	route = m.LastResponse.Route
	p = regPkgs.Get(route)
	if p == nil {
		return nil, route, false, ErrMissingPackage
	}
	return p, route, true, nil
}

func callPkg(pw *pkg.PkgWrapper, m *dt.Msg, followup bool) (*dt.RespMsg,
	error) {
	reply := &dt.RespMsg{}
	log.WithField("pkg", pw.P.Config.Name).Infoln("sending input")
	c := strings.Title(pw.P.Config.Name)
	// with fixed gob encoding this will not be necessary
	m.LastResponse = nil
	// TODO is this OR condition really necessary?
	if followup {
		log.WithField("pkg", pw.P.Config.Name).Infoln("follow up")
		c += ".FollowUp"
	} else {
		log.WithField("pkg", pw.P.Config.Name).Infoln("first run")
		c += ".Run"
	}
	// TODO pass LastResponse directly to packages via rpc gob encoding,
	// removing the need to nil this out and then look it up again in the
	// package
	m.LastResponse = nil
	if err := pw.RPCClient.Call(c, m, reply); err != nil {
		log.WithField("pkg", pw.P.Config.Name).Errorln(
			"invalid response", err)
		return reply, err
	}
	return reply, nil
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

func (am atomicMap) Get(k string) bool {
	var b bool
	am.mutex.Lock()
	b = am.words[k]
	am.mutex.Unlock()
	runtime.Gosched()
	return b
}

func (am atomicMap) Set(k string, v bool) {
	am.mutex.Lock()
	am.words[k] = v
	am.mutex.Unlock()
	runtime.Gosched()
}
