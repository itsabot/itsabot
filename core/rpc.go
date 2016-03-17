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
	"github.com/itsabot/abot/shared/plugin"
	"github.com/jmoiron/sqlx"
)

// Abot is defined to use in RPC communication
type Abot int

// ErrMissingPlugin denotes that Abot could find neither a plugin with
// matching triggers for a user's message nor any prior plugin used. This is
// most commonly seen on first run if the user's message doesn't initially
// trigger a plugin.
var ErrMissingPlugin = errors.New("missing plugin")

// BootRPCServer starts the rpc for Abot core in a go routine and returns the
// server address.
func BootRPCServer() (abot *Abot, addr string, err error) {
	log.Debug("booting abot core rpc server")
	abot = new(Abot)
	if err = rpc.Register(abot); err != nil {
		return abot, "", err
	}
	var ln net.Listener
	if ln, err = net.Listen("tcp", ":0"); err != nil {
		return abot, "", err
	}
	addr = ln.Addr().String()
	go func() {
		for {
			var conn net.Conn
			conn, err = ln.Accept()
			if err != nil {
				log.Debug("could not accept rpc", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
	return abot, addr, err
}

// BootDependencies executes all binaries listed in "plugins.json". each
// dependencies is passed the rpc address of the ava core. it is expected that
// each dependency respond with there own rpc address when registering
// themselves with the ava core.
func BootDependencies(avaRPCAddr string) error {
	log.Debug("booting dependencies")
	p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "itsabot",
		"abot", "plugins.json")
	content, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}
	var conf pluginsConf
	if err = json.Unmarshal(content, &conf); err != nil {
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
		// This assumes plugins are installed with go install ./...
		cmd := exec.Command(name, "-coreaddr", avaRPCAddr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			return err
		}
	}
	return nil
}

// GetPlugin attempts to find a plugin and route for the given msg input if none
// can be found, it checks the database for the last route used and gets the
// plugin for that. If there is no previously used plugin, we return
// ErrMissingPlugin. The bool value return indicates whether this plugin is
// different from the last plugin used by the user.
func GetPlugin(db *sqlx.DB, m *dt.Msg) (*plugin.Wrapper, string, bool, error) {
	// First check if the user is missing. AKA, needs to be onboarded
	if m.User == nil {
		p := regPlugins.Get("onboard_onboard")
		if p == nil {
			log.Debug("missing required onboard plugin")
			return nil, "onboard_onboard", false, ErrMissingPlugin
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
	log.Debugf("found user's last route: %q\n", prevRoute)

	// Iterate over all command/object pairs and see if any plugin has been
	// registered for the resulting route
	for _, c := range m.StructuredInput.Commands {
		for _, o := range m.StructuredInput.Objects {
			route := strings.ToLower(c + "_" + o)
			log.Debug("searching for route", route)
			if p := regPlugins.Get(route); p != nil {
				// Found route. Return it
				return p, route, false, nil
			}
		}
	}

	// The user input didnt match any plugins. Lets see if the prevRoute
	// does
	if prevRoute != "" {
		log.Debug("checking prevRoute for pkg")
		if p := regPlugins.Get(prevRoute); p != nil {
			// Prev route matches a pkg! Return it
			return p, prevRoute, true, nil
		}
	}

	// Sadly, if we've reached this point, we are at a loss.
	log.Debug("could not match user input to any plugin")
	return nil, "", false, ErrMissingPlugin
}

// CallPlugin sends a plugin the user's preprocessed message. The followup bool
// dictates whether this is the first consecutive time the user has sent that
// plugin a message, or if the user is engaged in a conversation with the plugin.
// This difference enables plugins to respond differently--like reset state--
// when messaged for the first time in each new conversation.
func CallPlugin(pw *plugin.Wrapper, m *dt.Msg, followup bool) (pluginReply string,
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
// appropriate plugins. The map's key is the route in the form of
// command_object, e.g. "find_restaurant", and the Wrapper contains both the
// plugin and the RPC client used to communicate with it.
type pkgMap struct {
	pkgs  map[string]*plugin.Wrapper
	mutex *sync.Mutex
}

// pluginsConf holds the structure of the plugins.json configuration file.
type pluginsConf struct {
	Name         string
	Version      string
	Dependencies map[string]string
}

// regPlugins initializes a pkgMap and holds it in global memory, which works OK
// given pkgMap is an atomic, thread-safe map.
var regPlugins = pkgMap{
	pkgs:  make(map[string]*plugin.Wrapper),
	mutex: &sync.Mutex{},
}

// RegisterPlugin enables Abot to notify plugins when specific StructuredInput
// is encountered matching triggers set in the plugins themselves. Note that
// plugins will only listen when ALL criteria are met and that there's no
// support currently for duplicate routes (e.g. "find_restaurant" leading to
// either one of two plugins).
func (t *Abot) RegisterPlugin(p *plugin.Plugin, _ *string) error {
	log.Debug("registering", p.Config.Name, "at", p.Config.PluginRPCAddr)
	client, err := rpc.Dial("tcp", p.Config.PluginRPCAddr)
	if err != nil {
		return err
	}
	for _, c := range p.Trigger.Commands {
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + "_" + o)
			if regPlugins.Get(s) != nil {
				log.Info("found duplicate plugin or trigger",
					p.Config.Name, "on", s)
			}
			regPlugins.Set(s, &plugin.Wrapper{P: p, RPCClient: client})
		}
	}
	return nil
}

// Get is a thread-safe, locking way to access the values of a pkgMap.
func (pm pkgMap) Get(k string) *plugin.Wrapper {
	var pw *plugin.Wrapper
	pm.mutex.Lock()
	pw = pm.pkgs[k]
	pm.mutex.Unlock()
	runtime.Gosched()
	return pw
}

// Set is a thread-safe, locking way to set the values of a pkgMap.
func (pm pkgMap) Set(k string, v *plugin.Wrapper) {
	pm.mutex.Lock()
	pm.pkgs[k] = v
	pm.mutex.Unlock()
	runtime.Gosched()
}
