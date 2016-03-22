package core

import (
	"database/sql"
	"runtime"
	"strings"
	"sync"

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/jmoiron/sqlx"
)

// PluginJSON holds the plugins.json structure.
type PluginJSON struct {
	Dependencies map[string]string
}

// RegPlugins initializes a pkgMap and holds it in global memory, which works OK
// given pkgMap is an atomic, thread-safe map.
var RegPlugins = pkgMap{
	pkgs:  make(map[string]*dt.Plugin),
	mutex: &sync.Mutex{},
}

// AllPlugins contains a set of all registered plugins.
var AllPlugins = []*dt.Plugin{}

// pkgMap is a thread-safe atomic map that's used to route user messages to the
// appropriate plugins. The map's key is the route in the form of
// command_object, e.g. "find_restaurant".
type pkgMap struct {
	pkgs  map[string]*dt.Plugin
	mutex *sync.Mutex
}

// Get is a thread-safe, locking way to access the values of a pkgMap.
func (pm pkgMap) Get(k string) *dt.Plugin {
	var p *dt.Plugin
	pm.mutex.Lock()
	p = pm.pkgs[k]
	pm.mutex.Unlock()
	runtime.Gosched()
	return p
}

// Set is a thread-safe, locking way to set the values of a pkgMap.
func (pm pkgMap) Set(k string, v *dt.Plugin) {
	pm.mutex.Lock()
	pm.pkgs[k] = v
	pm.mutex.Unlock()
	runtime.Gosched()
}

// CallPlugin sends a plugin the user's preprocessed message. The followup bool
// dictates whether this is the first consecutive time the user has sent that
// plugin a message, or if the user is engaged in a conversation with the
// plugin. This difference enables plugins to respond differently--like reset
// state--when messaged for the first time in each new conversation.
func CallPlugin(p *dt.Plugin, in *dt.Msg, followup bool) string {
	var reply string
	if p == nil {
		return reply
	}
	var err error
	if followup {
		reply, err = p.FollowUp(in)
	} else {
		reply, err = p.Run(in)
	}
	if err != nil {
		log.Debug(err)
	}
	return reply
}

// GetPlugin attempts to find a plugin and route for the given msg input if none
// can be found, it checks the database for the last route used and gets the
// plugin for that. If there is no previously used plugin, we return
// ErrMissingPlugin. The bool value return indicates whether this plugin is
// different from the last plugin used by the user.
func GetPlugin(db *sqlx.DB, m *dt.Msg) (*dt.Plugin, string, bool, error) {
	// First check if the user is missing. AKA, needs to be onboarded
	if m.User == nil {
		p := RegPlugins.Get("onboard_onboard")
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
			if p := RegPlugins.Get(route); p != nil {
				// Found route. Return it
				return p, route, false, nil
			}
		}
	}

	// The user input didn't match any plugins. Lets see if the prevRoute
	// does
	if prevRoute != "" {
		log.Debug("checking prevRoute for pkg")
		if p := RegPlugins.Get(prevRoute); p != nil {
			// Prev route matches a pkg! Return it
			return p, prevRoute, true, nil
		}
	}

	// Sadly, if we've reached this point, we are at a loss.
	log.Debug("could not match user input to any plugin")
	return nil, "", false, ErrMissingPlugin
}
