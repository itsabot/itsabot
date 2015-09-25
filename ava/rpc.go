package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/pkg"
)

type Ava int

type packageRegistration struct {
	port int
	sis  []datatypes.StructuredInput
}

var packages map[string]packageRegistration

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

func (t *Ava) RegisterPackage(r pkg.Registration, reply *error) error {
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

func (t *Ava) RespondTo(l pkg.Listener, reply *error) error {
	// TODO
	return nil
}
