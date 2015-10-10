package pkg

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/shared/datatypes"
)

type PkgWrapper struct {
	P         *Pkg
	RPCClient *rpc.Client
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
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p.Config.Port+1))
	if err != nil {
		log.Fatalln("rpc listen:", err, p.Config.Name)
	}
	if err := rpc.Register(pkgT); err != nil {
		log.Fatalln(err, p.Config.Name)
	}
	client, err = rpc.Dial("tcp", ":4001")
	if err != nil {
		return err
	}
	var notused error
	log.Println("calling register", p.Config.Name)
	err = client.Call("Ava.RegisterPackage", p, &notused)
	if err != nil {
		log.Println("err: registering package", p.Config.Name, err)
		return err
	}
	log.Println("connected with ava", p.Config.Name)
	if err = connectDB(); err != nil {
		return err
	}
	log.Println("connected with database", p.Config.Name)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go rpc.ServeConn(conn)
	}
	return nil
}

func connectDB() error {
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=egtann dbname=ava sslmode=disable")
	}
	if err != nil {
		return err
	}
	return nil
}
