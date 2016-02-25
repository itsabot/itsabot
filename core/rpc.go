package core

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/pkg"
	"github.com/jmoiron/sqlx"
)

// Abot is defined to use in RPC communication
type Abot int

// ErrMissingPackage denotes that Abot could find neither a package with
// matching triggers for a user's message nor any prior package used. This is
// most commonly seen on first run if the user's message doesn't initially
// trigger a package.
var ErrMissingPackage = errors.New("missing package")

// BootRPCServer starts the rpc for Abot core in a go routine and returns the
// server address
func BootRPCServer() (addr string, err error) {
	log.Debug("booting abot core rpc server")
	abot := new(Abot)
	if err = rpc.Register(abot); err != nil {
		return
	}
	var ln net.Listener
	if ln, err = net.Listen("tcp", ":0"); err != nil {
		return
	}
	addr = ln.Addr().String()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Debug("could not accept rpc", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
	return addr, err
}

// BootDependencies executes all binaries listed in "packages.json". each
// dependencies is passed the rpc address of the ava core. it is expected that
// each dependency respond with there own rpc address when registering
// themselves with the ava core.
func BootDependencies(avaRPCAddr string) error {
	log.Debug("booting dependencies")
	content, err := ioutil.ReadFile("packages.json")
	if err != nil {
		return err
	}
	var conf packagesConf
	if err := json.Unmarshal(content, &conf); err != nil {
		return err
	}
	for name, version := range conf.Dependencies {
		_, name = filepath.Split(name)
		if version == "*" {
			name += "-master"
		} else {
			name += "-" + version
		}
		log.Debug("booting", name)
		// This assumes packages are installed with go install ./...
		cmd := exec.Command(name, "-coreaddr", avaRPCAddr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			return err
		}
	}
	return nil
}

// GetPkg attempts to find a package and route for the given msg input if none
// can be found, it checks the database for the last route used and gets the
// package for that. If there is no previously used package, we return
// ErrMissingPackage. The bool value return indicates whether this package is
// different from the last package used by the user.
func GetPkg(db *sqlx.DB, m *dt.Msg) (*pkg.PkgWrapper, string, bool, error) {
	// First check if the user is missing. AKA, needs to be onboarded
	if m.User == nil {
		p := regPkgs.Get("onboard_onboard")
		if p == nil {
			log.Debug("missing required onboard package")
			return nil, "onboard_onboard", false, ErrMissingPackage
		}
		return p, "onboard_onboard", true, nil
	}

	// First we look for a previously used route. we have to do this in
	// any case to check if the users pkg/route has changed, so why not now?
	log.Debug("getting last route")
	prevRoute, err := m.GetLastRoute(db)
	if err != nil && err != sql.ErrNoRows {
		return nil, "", false, err
	}
	log.Debug("found user's last route: %q\n", prevRoute)

	// Iterate over all command/object pairs and see if any package has been
	// registered for the resulting route
	for _, c := range m.StructuredInput.Commands {
		for _, o := range m.StructuredInput.Objects {
			route := strings.ToLower(c + "_" + o)
			log.Debug("searching for route", route)
			if p := regPkgs.Get(route); p != nil {
				// Found route. Return it
				return p, route, false, nil
			}
		}
	}

	// The user input didnt match any packages. Lets see if the prevRoute
	// does
	if prevRoute != "" {
		log.Debug("checking prevRoute for pkg")
		if p := regPkgs.Get(prevRoute); p != nil {
			// Prev route matches a pkg! Return it
			return p, prevRoute, true, nil
		}
	}

	// Sadly, if we've reached this point, we are at a loss.
	log.Debug("could not match user input to any package")
	return nil, "", false, ErrMissingPackage
}

// CallPkg sends a package the user's preprocessed message. The followup bool
// dictates whether this is the first consecutive time the user has sent that
// package a message, or if the user is engaged in a conversation with the pkg.
// This difference enables packages to respond differently--like reset state--
// when messaged for the first time in each new conversation.
func CallPkg(pw *pkg.PkgWrapper, m *dt.Msg, followup bool) (packageReply string,
	err error) {

	tmp := ""
	reply := &tmp
	if pw == nil {
		return *reply, nil
	}
	log.Debug("sending input to", pw.P.Config.Name)
	c := strings.Title(pw.P.Config.Name)
	if followup {
		c += ".FollowUp"
	} else {
		c += ".Run"
	}
	if err := pw.RPCClient.Call(c, m, reply); err != nil {
		return *reply, err
	}
	return *reply, nil
}

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

// RegisterPackage enables Abot to notify packages when specific StructuredInput
// is encountered matching triggers set in the packages themselves. Note that
// packages will only listen when ALL criteria are met and that there's no
// support currently for duplicate routes (e.g. "find_restaurant" leading to
// either one of two packages).
func (t *Abot) RegisterPackage(p *pkg.Pkg, reply *string) error {
	log.Debug("registering", p.Config.Name, "at", p.Config.PkgRPCAddr)
	client, err := rpc.Dial("tcp", p.Config.PkgRPCAddr)
	if err != nil {
		return err
	}
	for _, c := range p.Trigger.Commands {
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + "_" + o)
			if regPkgs.Get(s) != nil {
				log.Info("found duplicate package or trigger",
					p.Config.Name, "on", s)
			}
			regPkgs.Set(s, &pkg.PkgWrapper{P: p, RPCClient: client})
		}
	}
	return nil
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
