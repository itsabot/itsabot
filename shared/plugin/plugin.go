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
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/nlp"
	"github.com/jmoiron/sqlx"
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
		return &dt.Plugin{}, ErrMissingTrigger
	}
	if fns == nil || fns.Run == nil || fns.FollowUp == nil {
		return &dt.Plugin{}, ErrMissingPluginFns
	}
	// Read plugin.json, unmarshal into struct
	p := filepath.Join(os.Getenv("GOPATH"), "src", url, "plugin.json")
	contents, err := ioutil.ReadFile(p)
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
	db, err := connectDB()
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
	}
	if err = RegisterPlugin(plg); err != nil {
		return nil, err
	}
	return plg, nil
}

// connectDB opens a connection to the database.
func connectDB() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("ABOT_ENV") == "production" {
		db, err = sqlx.Connect("postgres",
			os.Getenv("ABOT_DATABASE_URL"))
	} else if os.Getenv("ABOT_ENV") == "test" {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot_test sslmode=disable")
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot sslmode=disable")
	}
	return db, err
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
	return nil
}
