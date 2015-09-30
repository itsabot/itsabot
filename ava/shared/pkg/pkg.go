package pkg

import (
	"errors"
	"net/rpc"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PkgWrapper struct {
	P         *Pkg
	RPCClient *rpc.Client
	Logger    *log.Entry
}

// Pkg holds config options for any Ava package. Name must be globally unique
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Pkg struct {
	Config  PkgConfig
	Trigger *datatypes.StructuredInput
}

type PkgConfig struct {
	Name          string
	ServerAddress string
	Port          int
}

type Ava int

var client *rpc.Client
var db *sqlx.DB
var (
	ErrMissingPackageName = errors.New("missing package name")
	ErrMissingPort        = errors.New("missing package port")
	ErrMissingTrigger     = errors.New("missing package trigger")
)

func NewPackage(name, serverAddr string, port int,
	trigger *datatypes.StructuredInput) (*Pkg, error) {

	if len(name) == 0 {
		return &Pkg{}, ErrMissingPackageName
	}
	if port == 0 {
		return &Pkg{}, ErrMissingPort
	}
	if trigger == nil {
		return &Pkg{}, ErrMissingTrigger
	}
	c := PkgConfig{
		Name:          name,
		Port:          port,
		ServerAddress: serverAddr,
	}
	return &Pkg{Config: c, Trigger: trigger}, nil
}

// Register with Ava to begin communicating over RPC.
func (p *Pkg) Register() error {
	if os.Getenv("AVA_ENV") == "production" {
		log.SetLevel(log.WarnLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
	plog := log.WithField("package", p.Config.Name)
	var err error
	port := ":" + strconv.Itoa(p.Config.Port)
	client, err = rpc.Dial("tcp", p.Config.ServerAddress+port)
	if err != nil {
		return err
	}
	var notused error
	plog.Debug("registering with ava")
	err = client.Call("Ava.RegisterPackage", p, &notused)
	if err != nil && err.Error() != "gob: type rpc.Client has no exported fields" {
		plog.Error(err.Error())
		return err
	}
	plog.Debug("connected with ava")
	if err = connectDB(); err != nil {
		return err
	}
	plog.Debug("connected with database")
	plog.Info("loaded")
	return nil
}

func connectDB() error {
	var err error
	db, err = sqlx.Connect("postgres",
		"user=egtann dbname=ava sslmode=disable")
	if err != nil {
		return err
	}
	return nil
}
