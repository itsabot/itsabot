package pkg

import (
	"errors"
	"net"
	"net/rpc"
	"os"

	log "github.com/Sirupsen/logrus"
	"itsabot.org/abot/shared/datatypes"
	"itsabot.org/abot/shared/nlp"
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
	log.SetLevel(log.DebugLevel)
	log.WithFields(log.Fields{"pkg": p.Config.Name}).Debugln("connecting")

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}
	p.Config.PkgRPCAddr = ln.Addr().String()

	if err := rpc.Register(pkgT); err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}

	client, err := rpc.Dial("tcp", p.Config.CoreRPCAddr)
	if err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}

	if err := client.Call("Ava.RegisterPackage", p, nil); err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln("calling", err)
	}

	log.WithField("pkg", p.Config.Name).Debugln("connected")

	db, err = ConnectDB()
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln("connectDB", err)
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.WithField("pkg", p.Config.Name).Fatalln(err)
		}
		go rpc.ServeConn(conn)
	}
	return nil
}

func ConnectDB() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=ava sslmode=disable")
	}
	if err != nil {
		log.WithFields(log.Fields{
			"fn": "ConnectDB",
		}).Errorln(err)
	}
	return db, err
}
