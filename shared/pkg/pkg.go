package pkg

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

type PkgWrapper struct {
	P         *Pkg
	RPCClient *rpc.Client
}

// Pkg holds config options for any Ava package. Name must be globally unique.
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Pkg struct {
	Config  PkgConfig
	Vocab   *dt.Vocab
	Trigger *nlp.StructuredInput
}

type PkgConfig struct {
	Name        string
	Route       string
	CoreRPCAddr string
	PkgRPCAddr  string
}

type Ava int

var client *rpc.Client
var db *sqlx.DB
var (
	ErrMissingPackageName = errors.New("missing package name")
	ErrMissingPort        = errors.New("missing package port")
	ErrMissingTrigger     = errors.New("missing package trigger")
)

func NewPackage(name, coreRPCAddr string,
	trigger *nlp.StructuredInput) (*Pkg, error) {

	if len(name) == 0 {
		return &Pkg{}, ErrMissingPackageName
	}
	if trigger == nil {
		return &Pkg{}, ErrMissingTrigger
	}

	c := PkgConfig{
		Name:        name,
		CoreRPCAddr: coreRPCAddr,
	}
	return &Pkg{Config: c, Trigger: trigger}, nil
}

// Register with Ava to begin communicating over RPC.
func (p *Pkg) Register(pkgT interface{}) error {
	log.Debug("connecting to", p.Config.Name)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	p.Config.PkgRPCAddr = ln.Addr().String()

	if err := rpc.Register(pkgT); err != nil {
		return err
	}

	client, err := rpc.Dial("tcp", p.Config.CoreRPCAddr)
	if err != nil {
		return err
	}

	if err := client.Call("Abot.RegisterPackage", p, nil); err != nil {
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
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("ABOT_DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=ava sslmode=disable")
	}
	return db, err
}
