package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"itsabot.org/abot/shared/datatypes"
	"itsabot.org/abot/shared/pkg"
)

type Ava int

// pkgMap is a thread-safe atomic map that's used to route user messages to the
// appropriate packages. The map's key is the route in the form of
// command_object, e.g. "find_restaurant", and the PkgWrapper contains both the
// package and the RPC client used to communicate with it.
type pkgMap struct {
	pkgs  map[string]*pkg.PkgWrapper
	mutex *sync.Mutex
}

// packagesConf holds the structure of the packages.json configuration file.
type packagesConf struct {
	Name         string
	Version      string
	Dependencies map[string]string
}

// regPkgs initializes a pkgMap and holds it in global memory, which works OK
// given pkgMap is an atomic, thread-safe map.
var regPkgs = pkgMap{
	pkgs:  make(map[string]*pkg.PkgWrapper),
	mutex: &sync.Mutex{},
}

// RegisterPackage enables Ava to notify packages when specific StructuredInput
// is encountered matching triggers set in the packages themselves. Note that
// packages will only listen when ALL criteria are met and that there's no
// support currently for duplicate routes (e.g. "find_restaurant" leading to
// either one of two packages).
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
		for _, o := range p.Trigger.Objects {
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
	return nil
}

// getPkg attempts to find a package and route for the given msg input if none
// can be found, it checks the database for the last route used and gets the
// package for that. If there is no previously used package, we return
// ErrMissingPackage. The bool value return indicates whether this package is
// different from the last package used by the user.
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

// callPkg sends a package the user's preprocessed message. The followup bool
// dictates whether this is the first consecutive time the user has sent that
// package a message, or if the user is engaged in a conversation with the pkg.
// This difference enables packages to respond differently--like reset state--
// when messaged for the first time in each new conversation.
func callPkg(pw *pkg.PkgWrapper, m *dt.Msg, followup bool) (packageReply string,
	err error) {

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

// bootDependencies executes all binaries listed in "packages.json". each
// dependencies is passed the rpc address of the ava core. it is expected that
// each dependency respond with there own rpc address when registering
// themselves with the ava core.
func bootDependencies(avaRPCAddr string) {
	log.WithFields(log.Fields{
		"ava_core_addr": avaRPCAddr,
	}).Debugln("booting dependencies")
	content, err := ioutil.ReadFile("packages.json")
	if err != nil {
		log.Fatalln("reading packages.json", err)
	}
	var conf packagesConf
	if err := json.Unmarshal(content, &conf); err != nil {
		log.Fatalln("err: unmarshaling packages", err)
	}
	for name, version := range conf.Dependencies {
		_, name = filepath.Split(name)
		if version == "*" {
			name += "-master"
		} else {
			name += "-" + version
		}
		log.WithFields(log.Fields{"pkg": name}).Debugln("booting")
		// This assumes packages are installed with go install ./...
		cmd := exec.Command(name, "-coreaddr", avaRPCAddr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			log.WithFields(log.Fields{
				"pkg": name,
			}).Fatalln(err)
		}
	}
}

// Get is a thread-safe, locking way to access the values of a pkgMap.
func (pm pkgMap) Get(k string) *pkg.PkgWrapper {
	var pw *pkg.PkgWrapper
	pm.mutex.Lock()
	pw = pm.pkgs[k]
	pm.mutex.Unlock()
	runtime.Gosched()
	return pw
}

// Set is a thread-safe, locking way to set the values of a pkgMap.
func (pm pkgMap) Set(k string, v *pkg.PkgWrapper) {
	pm.mutex.Lock()
	pm.pkgs[k] = v
	pm.mutex.Unlock()
	runtime.Gosched()
}
