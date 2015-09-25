package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/avabot/avabot/types"
)

type Ava int
type packageRegistration struct {
	port int
	sis  []types.StructuredInput
}

var packages map[string]packageRegistration

// Listener includes the name of the package and the structured input to listen
// for. StructuredInput is case insensitive.
type Listener struct {
	Name string
	SI   *types.StructuredInput
}

// Registration
type Registration struct {
	Name string
	Port string
}

func bootRPCServer() {
	ava := new(Ava)
	rpc.Register(ava)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":4001")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func (t *Ava) RegisterPackage(r Registration, reply *error) error {
	// TODO
	// Keep track of packages. Ensure no duplicates.
	/*
		pr := packageRegistration{}
		if v := packages[r.Name]; v != pr {
			return errors.New()
		}
	*/
	return nil
}

func (t *Ava) RespondTo(l Listener, reply *error) error {
	// TODO
	return nil
}
