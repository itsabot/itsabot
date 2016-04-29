// Package plugin enables plugins to register with Abot and connect to the
// database.
package plugin

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/nlp"
	_ "github.com/lib/pq" // Import the pq PostgreSQL driver
)

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
		return nil, ErrMissingTrigger
	}
	if fns == nil || fns.Run == nil || fns.FollowUp == nil {
		return nil, ErrMissingPluginFns
	}

	// Read plugin.json data from within plugins.go, unmarshal into struct
	p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "itsabot",
		"abot", "plugins.go")
	fi, err := os.OpenFile(p, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := fi.Close(); err != nil {
			log.Info("failed to close file", fi.Name())
			return
		}
	}()
	var found bool
	var data string
	scn := bufio.NewScanner(fi)
	for scn.Scan() {
		t := scn.Text()
		if !found && t != url {
			continue
		} else if t == url {
			found = true
			continue
		} else if len(t) >= 1 && t[0] == '}' {
			data += t
			break
		}
		data += t
	}
	if err := scn.Err(); err != nil {
		return nil, err
	}
	plg := &dt.Plugin{
		Trigger:   trigger,
		PluginFns: fns,
		Events: &dt.PluginEvents{
			PostReceive:    func(cmd *string) {},
			PreProcessing:  func(cmd *string, u *dt.User) {},
			PostProcessing: func(in *dt.Msg) {},
			PostResponse:   func(in *dt.Msg, resp *string) {},
		},
	}
	c := dt.PluginConfig{}
	if len(data) > 0 {
		if err = json.Unmarshal([]byte(data), &c); err != nil {
			log.Info("error here!")
			return nil, err
		}
		if len(c.Name) == 0 {
			return nil, ErrMissingPluginName
		}
	}
	plg.Config = c
	db, err := core.ConnectDB()
	if err != nil {
		return nil, err
	}
	plg.DB = db
	l := log.New(c.Name)
	l.SetDebug(os.Getenv("ABOT_DEBUG") == "true")
	plg.Log = l
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
