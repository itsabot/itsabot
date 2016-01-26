package pkg

import (
	"errors"
	"net"
	"net/rpc"
	"os"
	"strconv"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
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
	Name          string
	ServerAddress string
	Route         string
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

func NewPackage(name string, port int, trigger *nlp.StructuredInput) (
	*Pkg, error) {
	return NewPackageWithServer(name, "", port, trigger)
}

func NewPackageWithServer(name, serverAddr string, port int,
	trigger *nlp.StructuredInput) (*Pkg, error) {
	if len(name) == 0 {
		return &Pkg{}, ErrMissingPackageName
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
func (p *Pkg) Register(pkgT interface{}) error {
	log.SetLevel(log.DebugLevel)
	log.WithFields(log.Fields{
		"port": p.Config.Port + 1,
		"pkg":  p.Config.Name,
	}).Debugln("connecting")
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p.Config.Port+1))
	if err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}
	if err := rpc.Register(pkgT); err != nil {
		log.WithField("pkg", p.Config.Name).Fatalln(err)
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln(err)
		return err
	}
	client, err = rpc.Dial("tcp", ":"+strconv.Itoa(port+1))
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln(err)
		return err
	}
	var notused string
	err = client.Call("Ava.RegisterPackage", p, &notused)
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln("calling", err)
		return err
	}
	log.WithField("pkg", p.Config.Name).Debugln("connected")
	db, err = ConnectDB()
	if err != nil {
		log.WithField("pkg", p.Config.Name).Errorln("connectDB", err)
		return err
	}
	for {
		conn, err := l.Accept()
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
