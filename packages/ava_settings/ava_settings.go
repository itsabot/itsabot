package main

import (
	"flag"
	"math/rand"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
	"github.com/avabot/ava/shared/pkg"
)

type Settings string

var vocab dt.Vocab
var db *sqlx.DB
var p *pkg.Pkg
var sm *dt.StateMachine
var l *log.Entry

const pkgName string = "settings"

func main() {
	var coreaddr string
	flag.StringVar(&coreaddr, "coreaddr", "",
		"Port used to communicate with Ava.")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	l = log.WithFields(log.Fields{"pkg": pkgName})
	rand.Seed(time.Now().UnixNano())
	var err error
	db, err = pkg.ConnectDB()
	if err != nil {
		l.Fatalln(err)
	}
	trigger := &nlp.StructuredInput{
		Commands: []string{"change", "modify", "switch", "alter", "add",
			"remove", "delete"},
		Objects: []string{"card", "address", "calendar"},
	}
	p, err = pkg.NewPackage(pkgName, coreaddr, trigger)
	if err != nil {
		l.Fatalln("building", err)
	}
	p.Vocab = dt.NewVocab(
		// TODO change handlers to use triggers
		dt.VocabHandler{
			Fn:       kwAddCard,
			WordType: "Object",
			Words:    []string{"card"},
		},
		dt.VocabHandler{
			Fn:       kwChangeCard,
			WordType: "Command",
			Words: []string{"change", "modify", "delete", "switch",
				"alter"},
		},
	)
	sm, err = dt.NewStateMachine(pkgName)
	if err != nil {
		l.Errorln(err)
		return
	}
	sm.SetStates([]dt.State{})
	sm.SetDBConn(db)
	sm.SetLogger(l)
	settings := new(Settings)
	if err := p.Register(settings); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Settings) Run(in *dt.Msg, resp *string) error {
	sm.Reset(in)
	return t.FollowUp(in, resp)
}

func (t *Settings) FollowUp(in *dt.Msg, resp *string) error {
	*resp = p.Vocab.HandleKeywords(in)
	if len(*resp) == 0 {
		*resp = sm.Next(in)
	}
	return nil
}

func kwAddCard(in *dt.Msg, _ int) string {
	sm.SetStates(
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					return "Sure. You can add your card here: https://avabot.co/?/cards/new"
				}, OnInput: func(in *dt.Msg) {
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return true, ""
				},
			},
		},
	)
	return sm.Next(in)
}

func kwChangeCard(in *dt.Msg, _ int) string {
	sm.SetStates(
		[]dt.State{
			{
				OnEntry: func(in *dt.Msg) string {
					return "Sure. You can change your cards here: https://avabot.co/?/profile"
				}, OnInput: func(in *dt.Msg) {
				},
				Complete: func(in *dt.Msg) (bool, string) {
					return true, ""
				},
			},
		},
	)
	return sm.Next(in)
}
