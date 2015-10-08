package pkg

import (
	"errors"
	"net"
	"net/rpc"
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

func NewPackage(name string, port int, trigger *datatypes.StructuredInput) (
	*Pkg, error) {
	return NewPackageWithServer(name, "", port, trigger)
}

func NewPackageWithServer(name, serverAddr string, port int,
	trigger *datatypes.StructuredInput) (*Pkg, error) {
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
	plog := log.WithField("package", p.Config.Name)
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p.Config.Port+1))
	if err != nil {
		plog.Fatalln("rpc listen:", err)
	}
	if err := rpc.Register(pkgT); err != nil {
		plog.Fatal(err)
	}
	port := ":" + strconv.Itoa(p.Config.Port+1)
	client, err = rpc.Dial("tcp", p.Config.ServerAddress+port)
	if err != nil {
		return err
	}
	var notused error
	err = client.Call("Ava.RegisterPackage", p, &notused)
	if err != nil &&
		err.Error() != "gob: type rpc.Client has no exported fields" {
		plog.Error(err.Error())
		return err
	}
	plog.Debug("connected with ava")
	if err = connectDB(); err != nil {
		return err
	}
	plog.Debug("connected with database")
	plog.Info("loaded")
	for {
		conn, err := l.Accept()
		if err != nil {
			plog.Fatal(err)
		}
		go rpc.ServeConn(conn)
	}
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
