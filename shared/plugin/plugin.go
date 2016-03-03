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
	_ "github.com/lib/pq"
)

type PluginWrapper struct {
	P         *Plugin
	RPCClient *rpc.Client
}

// Plugin holds config options for any Abot plugin. Name must be globally unique.
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Plugin struct {
	Config  PluginConfig
	Vocab   *dt.Vocab
	Trigger *nlp.StructuredInput
}

type PluginConfig struct {
	Name          string
	Route         string
	CoreRPCAddr   string
	PluginRPCAddr string
}

type Abot int

var client *rpc.Client
var db *sqlx.DB
var (
	ErrMissingPluginName = errors.New("missing plugin name")
	ErrMissingPort       = errors.New("missing plugin port")
	ErrMissingTrigger    = errors.New("missing plugin trigger")
)

func NewPlugin(name, coreRPCAddr string, trigger *nlp.StructuredInput) (*Plugin, error) {

	if len(name) == 0 {
		return &Plugin{}, ErrMissingPluginName
	}
	if trigger == nil {
		return &Plugin{}, ErrMissingTrigger
	}

	c := PluginConfig{
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

	if err := rpc.Register(pluginT); err != nil {
		return err
	}

	client, err := rpc.Dial("tcp", p.Config.CoreRPCAddr)
	if err != nil {
		return err
	}

	if err := client.Call("Abot.RegisterPlugin", p, nil); err != nil {
		return err
	}

	log.Debug("connected to", p.Config.Name)

	db, err = ConnectDB()
	if err != nil {
		log.Debug("could not connect to database")
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Debug("could not accept connections for",
				p.Config.Name, ", ", err)
		}
		go rpc.ServeConn(conn)
	}
	return nil
}

func ConnectDB() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("ABOT_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("ABOT_DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot sslmode=disable")
	}
	return db, err
}
