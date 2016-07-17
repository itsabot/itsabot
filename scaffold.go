package main

import (
	"os"

	"github.com/itsabot/abot/core"
)

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

func serverAbotEnv(name, curDir string) string {
	return `PORT=` + os.Getenv("PORT") + `
ABOT_ENV=development
ABOT_PATH="` + curDir + `"
ABOT_DATABASE_URL="` + core.DBConnectionString(name) + `"
ABOT_SECRET=` + core.RandAlphaNumSeq(64) + `
ABOT_URL=http://localhost:$PORT`
}
