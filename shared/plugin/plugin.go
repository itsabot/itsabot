// Package plugin enables plugins to register with Abot and connect to the
// database.
package plugin

import (
	"errors"
	"net"
	"net/rpc"
	"os"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/nlp"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Import the pq PostgreSQL driver
)

// Wrapper wraps a plugin with an open connection to an RPC client.
type Wrapper struct {
	P         *Plugin
	RPCClient *rpc.Client
}

// Plugin holds config options for any Abot plugin. Name must be globally unique.
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Plugin struct {
	Config  Config
	Vocab   *dt.Vocab
	Trigger *nlp.StructuredInput
}

// Config holds options for a plugin.
type Config struct {
	Name          string
	Route         string
	CoreRPCAddr   string
	PluginRPCAddr string
}

var (
	// ErrMissingPluginName is returned when a plugin name is expected, but
	// but a blank name is provided.
	ErrMissingPluginName = errors.New("missing plugin name")

	// ErrMissingTrigger is returned when a trigger is expected but none
	// were found.
	ErrMissingTrigger = errors.New("missing plugin trigger")
)

// New builds a Plugin with its trigger, RPC, and name configured.
func New(name, coreRPCAddr string, trigger *nlp.StructuredInput) (*Plugin,
	error) {

	if len(name) == 0 {
		return &Plugin{}, ErrMissingPluginName
	}
	if trigger == nil {
		return &Plugin{}, ErrMissingTrigger
	}
	c := Config{
		Name:        name,
		CoreRPCAddr: coreRPCAddr,
	}
	return &Plugin{Config: c, Trigger: trigger}, nil
}

// Register with Abot to begin communicating over RPC.
func (p *Plugin) Register(pluginT interface{}) error {
	log.Debug("connecting to", p.Config.Name)
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	p.Config.PluginRPCAddr = ln.Addr().String()
	if err = rpc.Register(pluginT); err != nil {
		return err
	}
	client, err := rpc.Dial("tcp", p.Config.CoreRPCAddr)
	if err != nil {
		return err
	}
	if err = client.Call("Abot.RegisterPlugin", p, nil); err != nil {
		return err
	}
	log.Debug("connected to", p.Config.Name)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Debug("could not accept connections for",
				p.Config.Name, ", ", err)
		}
		go rpc.ServeConn(conn)
	}
}

// ConnectDB opens a connection to the database.
func ConnectDB() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("ABOT_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("ABOT_DATABASE_URL"))
	} else if os.Getenv("ABOT_ENV") == "test" {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot_test sslmode=disable")
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot sslmode=disable")
	}
	return db, err
}
