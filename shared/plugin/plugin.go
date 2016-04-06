// Package plugin enables plugins to register with Abot and connect to the
// database.
package plugin

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/nlp"
	_ "github.com/lib/pq" // Import the pq PostgreSQL driver
)

// PathError is thrown when GOPATH cannot be located
var PathError = errors.New("GOPATH env variable not set")

// ErrMissingPluginName is returned when a plugin name is expected, but
// but a blank name is provided.
var ErrMissingPluginName = errors.New("missing plugin name")

// ErrMissingTrigger is returned when a trigger is expected but none
// were found.
var ErrMissingTrigger = errors.New("missing plugin trigger")

// ErrMissingPluginFns is returned when plugin functions are expected but none
// were found.
var ErrMissingPluginFns = errors.New("missing plugin functions")

// New builds a Plugin with its trigger, RPC, and configuration settings from
// its plugin.json.
func New(url string, trigger *nlp.StructuredInput,
	fns *dt.PluginFns) (*dt.Plugin, error) {

	if trigger == nil {
		return &dt.Plugin{}, ErrMissingTrigger
	}
	if fns == nil || fns.Run == nil || fns.FollowUp == nil {
		return &dt.Plugin{}, ErrMissingPluginFns
	}
	// Read plugin.json, unmarshal into struct
	var contents []byte
	var err error
	path := os.Getenv("GOPATH")
	tokenizedPath := strings.Split(path, string(os.PathListSeparator))
	for _, subPath := range tokenizedPath {
		p := filepath.Join(subPath, "src", url, "plugin.json")
		if contents, err = ioutil.ReadFile(p); err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	c := dt.PluginConfig{}
	if err = json.Unmarshal(contents, &c); err != nil {
		return nil, err
	}
	if len(c.Name) == 0 {
		return nil, ErrMissingPluginName
	}
	db, err := core.ConnectDB()
	if err != nil {
		return nil, err
	}
	l := log.New(c.Name)
	l.SetDebug(os.Getenv("ABOT_DEBUG") == "true")
	plg := &dt.Plugin{
		Config:    c,
		Trigger:   trigger,
		DB:        db,
		Log:       log.New(c.Name),
		PluginFns: fns,
		Events: &dt.PluginEvents{
			PostReceive:    func(cmd *string) {},
			PreProcessing:  func(cmd *string, u *dt.User) {},
			PostProcessing: func(in *dt.Msg) {},
			PostResponse:   func(in *dt.Msg, resp *string) {},
		},
	}
	if err = RegisterPlugin(plg); err != nil {
		return nil, err
	}
	return plg, nil
}

// RegisterPlugin enables Abot to notify plugins when specific StructuredInput
// is encountered matching triggers set in the plugins themselves. Note that
// plugins will only listen when ALL criteria are met and that there's no
// support currently for duplicate routes (e.g. "find_restaurant" leading to
// either one of two plugins).
func RegisterPlugin(p *dt.Plugin) error {
	log.Debug("registering", p.Config.Name)
	for _, c := range p.Trigger.Commands {
		for _, o := range p.Trigger.Objects {
			s := strings.ToLower(c + "_" + o)
			if core.RegPlugins.Get(s) != nil {
				log.Info("found duplicate plugin or trigger",
					p.Config.Name, "on", s)
			}
			core.RegPlugins.Set(s, p)
		}
	}
	core.AllPlugins = append(core.AllPlugins, p)
	return nil
}
