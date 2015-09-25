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
	Name          string
	Port          string
	ServerAddress string
}

// Listener includes the name of the package and the structured input to listen
// for. StructuredInput is case insensitive.
type Listener struct {
	Name string
	SI   *datatypes.StructuredInput
}

// Registration
type Registration struct {
	Name string
	Port string
}

type Ava int

var client *rpc.Client
var pkgName string
var (
	ErrMissingPackageName = errors.New("missing package name")
	ErrMissingPort        = errors.New("missing package port")
)

func NewPackage(name, port, serverAddr string) (*Pkg, error) {
	if len(name) == 0 {
		return &Pkg{}, ErrMissingPackageName
	}
	if len(port) == 0 {
		return &Pkg{}, ErrMissingPort
	}
	return &Pkg{Name: name}, nil
}

// Register with Ava to begin communicating over RPC.
func (p *Pkg) Register() error {
	var err error
	var replyErr error
	client, err = rpc.DialHTTP("tcp", p.ServerAddress+":4001")
	if err != nil {
		return err
	}
	args := Registration{Name: p.Name, Port: p.Port}
	err = client.Call("Ava:RegisterPackage", args, &replyErr)
	if err != nil {
		return err
	}
	if replyErr != nil {
		return replyErr
	}
	bootRPCServer(p.Port)
	return nil
}

// RespondTo a specific StructuredInput, such as Command:Order && Object:Uber.
func (p *Pkg) RespondTo(si *datatypes.StructuredInput) error {
	var replyErr error
	args := Listener{Name: pkgName, SI: si}
	if err := client.Call("Ava:RespondTo", args, &replyErr); err != nil {
		return err
	}
	if replyErr != nil {
		return replyErr
	}
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
