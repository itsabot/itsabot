package main

func pluginScaffoldFile(dir, name string) string {
	return `package ` + name + `

import (
	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/plugin"
)

var p *dt.Plugin

const memKey = "firstToken"

func init() {
	var err error
	p, err = plugin.New("` + dir + `")
	if err != nil {
		log.Fatal("failed to build plugin ` + name + `", err)
	}
	plugin.SetKeywords(p,
		dt.KeywordHandler{
			Fn: kwDemo,
			Trigger: &dt.StructuredInput{
				Commands: []string{
					"show",
				},
				Objects: []string{
					"demo",
				},
			},
		},
	)
	plugin.SetStates(p, [][]dt.State{[]dt.State{
		{
			OnEntry: func(in *dt.Msg) string {
				return "This is a demo."
			},
			OnInput: func(in *dt.Msg) {
				if len(in.Tokens) == 0 {
					return
				}
				p.SetMemory(in, memKey, in.Tokens[0])
			},
			Complete: func(in *dt.Msg) (bool, string) {
				return p.HasMemory(in, memKey), "I didn't understand that."
			},
		},
	}})
	p.SM.SetOnReset(func(in *dt.Msg) {
		p.DeleteMemory(in, memKey)
	})
	if err = plugin.Register(p); err != nil {
		p.Log.Fatalf("failed to register plugin ` + name + `. %s", err)
	}
}

func kwDemo(in *dt.Msg) string {
	return "It worked! You typed: " + in.Sentence
}`
}

func pluginTestScaffoldFile(name string) string {
	return `package ` + name + `

import (
	"os"
	"testing"

	"github.com/itsabot/abot/shared/plugin"
	"github.com/julienschmidt/httprouter"
)

var r *httprouter.Router

func TestMain(m *testing.M) {
	r = plugin.TestPrepare()
	os.Exit(m.Run())
}

func TestKWDemo(t *testing.T) {
	seqTests := []string{
		"Show me a demo",
		"It worked!",
	}
	if len(seqTests) % 2 != 0 {
		t.Fatal("must have an even number of cases covering input -> expected")
	}
	for i := 0; i+i < len(seqTests); i += 2 {
		err := plugin.TestReq(r, seqTests[i], seqTests[i+1])
		if err != nil {
			t.Fatal(err)
		}
	}
}`
}
