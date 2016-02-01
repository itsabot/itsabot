package main

import (
	"database/sql"
	"net/rpc"
	"runtime"
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

var appVocab dt.AtomicMap

// RegisterPackage enables Ava to notify packages when specific StructuredInput
// is encountered. Note that packages will only listen when ALL criteria are met
func (t *Ava) RegisterPackage(p *pkg.Pkg, reply *string) error {
	log.WithFields(log.Fields{
		"pkg":  p.Config.Name,
		"addr": p.Config.PkgRPCAddr,
	}).Debugln("registering")
	client, err := rpc.Dial("tcp", p.Config.PkgRPCAddr)
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
			regPkgs.Set(s, &pkg.PkgWrapper{P: p, RPCClient: client})
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

// getPkg attempts to find a package and route for the given msg input if none
// can be found, it checks the database for the last route used and gets the
// package for that. If there is no previously used package, we return
// ErrMissingPackage. The bool value return indicates whether this package is
// different from the last package used by the user
func getPkg(m *dt.Msg) (*pkg.PkgWrapper, string, bool, error) {
	// First check if the user is missing. AKA, needs to be onboarded
	if m.User == nil {
		p := regPkgs.Get("onboard_onboard")
		if p == nil {
			log.Errorln("missing required onboard package")
			return nil, "onboard_onboard", false, ErrMissingPackage
		}
		return p, "onboard_onboard", true, nil
	}

	// First we look for a previously used route. we have to do this in
	// any case to check if the users pkg/route has changed, so why not now?
	log.Debugln("getting last route")
	prevRoute, err := m.GetLastRoute(db)
	if err != nil && err != sql.ErrNoRows {
		return nil, "", false, err
	}
	log.Debugf("user's last route: %q\n", prevRoute)

	// Iterate over all command/object pairs and see if any package has been
	// registered for the resulting route
	for _, c := range m.StructuredInput.Commands {
		for _, o := range m.StructuredInput.Objects {
			route := strings.ToLower(c + "_" + o)
			log.Debugln("searching route", route)
			if p := regPkgs.Get(route); p != nil {
				// Found route. Return it
				return p, route, false, nil
			}
		}
	}

	// The user input didnt match any packages. Lets see if the prevRoute
	// does
	if prevRoute != "" {
		log.Debugln("checking prevRoute for pkg")
		if p := regPkgs.Get(prevRoute); p != nil {
			// Prev route matches a pkg! Return it
			return p, prevRoute, true, nil
		}
	}

	// Sadly, if we've reached this point, we are at a loss.
	log.Warnln("could not match user input to any package")
	return nil, "", false, ErrMissingPackage
}

func callPkg(pw *pkg.PkgWrapper, m *dt.Msg, followup bool) (string, error) {
	tmp := ""
	reply := &tmp
	if pw == nil {
		return *reply, nil
	}
	log.WithField("pkg", pw.P.Config.Name).Infoln("sending input")
	c := strings.Title(pw.P.Config.Name)
	if followup {
		c += ".FollowUp"
	} else {
		c += ".Run"
	}
	if err := pw.RPCClient.Call(c, m, reply); err != nil {
		log.WithField("pkg", pw.P.Config.Name).Errorln(
			"invalid response", err)
		return *reply, err
	}
	return *reply, nil
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
