package core

import (
	"database/sql"
	"runtime"
	"strings"
	"sync"

	"github.com/dchest/stemmer/porter2"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/jmoiron/sqlx"
)

// PluginJSON holds the plugins.json structure.
type PluginJSON struct {
	Name         string
	Description  string
	Version      string
	ImportPath   string
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

// GetPlugin attempts to find a plugin and route for the given msg input if none
// can be found, it checks the database for the last route used and gets the
// plugin for that. If there is no previously used plugin, we return
// ErrMissingPlugin. The bool value return indicates whether this plugin is
// different from the last plugin used by the user.
func GetPlugin(db *sqlx.DB, m *dt.Msg) (p *dt.Plugin, route string, directroute,
	followup bool, err error) {

	// Iterate through all intents to see if any plugin has been registered
	// for the route
	for _, i := range m.StructuredInput.Intents {
		route = "__intent_" + strings.ToLower(i)
		log.Debug("searching for route", route)
		if p = RegPlugins.Get(route); p != nil {
			// Found route. Return it
			return p, route, true, false, nil
		}
	}

	// Iterate over all command/object pairs and see if any plugin has been
	// registered for the resulting route
	eng := porter2.Stemmer
	for _, c := range m.StructuredInput.Commands {
		c = eng.Stem(c)
		for _, o := range m.StructuredInput.Objects {
			route := strings.ToLower(c + "_" + eng.Stem(o))
			log.Debug("searching for route", route)
			if p = RegPlugins.Get(route); p != nil {
				// Found route. Return it
				return p, route, true, false, nil
			}
		}
	}

	// The user input didn't match any plugins. Let's see if the previous
	// route does
	log.Debug("getting last route")
	prevRoute, err := m.GetLastRoute(db)
	if err != nil && err != sql.ErrNoRows {
		return nil, "", false, false, err
	}
	log.Debugf("found user's last route: %q\n", prevRoute)
	if prevRoute != "" {
		if p = RegPlugins.Get(prevRoute); p != nil {
			// Prev route matches a pkg! Return it
			return p, prevRoute, false, true, nil
		}
	}

	// Sadly, if we've reached this point, we are at a loss.
	log.Debug("could not match user input to any plugin")
	return nil, "", false, false, ErrMissingPlugin
}
