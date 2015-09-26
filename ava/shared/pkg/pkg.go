package pkg

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/avabot/ava/shared/datatypes"
)

// Pkg holds config options for any Ava package. Name must be globally unique
// Port takes the format of ":1234". Note that the colon is significant.
// ServerAddress will default to localhost if left blank.
type Pkg struct {
	Config  PkgConfig
	Trigger *datatypes.StructuredInput
}

type PkgConfig struct {
	Name          string
	Port          string
	ServerAddress string
}

type Ava int

var client *rpc.Client
var (
	ErrMissingPackageName = errors.New("missing package name")
	ErrMissingPort        = errors.New("missing package port")
	ErrMissingTrigger     = errors.New("missing package trigger")
)

func NewPackage(name, port, serverAddr string,
	trigger *datatypes.StructuredInput) (*Pkg, error) {

	if len(name) == 0 {
		return &Pkg{}, ErrMissingPackageName
	}
	if len(port) == 0 {
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
	var err error
	client, err = rpc.DialHTTP("tcp", p.Config.ServerAddress+":4001")
	if err != nil {
		return err
	}
	var replyErr error
	err = client.Call("Ava:RegisterPackage", p, &replyErr)
	if err != nil {
		return err
	}
	if replyErr != nil {
		return replyErr
	}
	bootRPCServer(p.Config.Port)
	return nil
}

func bootRPCServer(port string) {
	ava := new(Ava)
	rpc.Register(ava)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}
