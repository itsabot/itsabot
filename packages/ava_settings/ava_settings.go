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
var l *log.Entry

const pkgName string = "settings"
const (
	stateInvalid int = iota
	stateAddCard
	stateChangeCard
)

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
	settings := new(Settings)
	if err := p.Register(settings); err != nil {
		l.Fatalln("registering", err)
	}
}

func (t *Settings) Run(in *dt.Msg, resp *string) error {
	sm := bootStateMachine(in)
	sm.SetOnReset(func(in *dt.Msg) {
		sm.SetMemory(in, "state", stateInvalid)
	})
	sm.SetMemory(in, "__state_entered", false)
	return handleInput(in, resp)
}

func (t *Settings) FollowUp(in *dt.Msg, resp *string) error {
	return handleInput(in, resp)
}

func handleInput(in *dt.Msg, resp *string) error {
	sm := bootStateMachine(in)
	sm.SetOnReset(func(in *dt.Msg) {
		sm.SetMemory(in, "state", stateInvalid)
	})
	*resp = p.Vocab.HandleKeywords(in)
	if len(*resp) == 0 {
		state := int(sm.GetMemory(in, "state").Int64())
		switch state {
		case stateAddCard:
			l.Debugln("setting state addCard")
			sm.SetStates(addCard)
		case stateChangeCard:
			l.Debugln("setting state changeCard")
			sm.SetStates(changeCard)
		default:
			l.Warnln("unrecognized state", state)
		}
		*resp = sm.Next(in)
	}
	return nil
}

func kwAddCard(in *dt.Msg, _ int) string {
	sm := bootStateMachine(in)
	sm.SetMemory(in, "state", stateAddCard)
	l.Warnln("kwAddCard hit")
	return ""
}

func kwChangeCard(in *dt.Msg, _ int) string {
	sm := bootStateMachine(in)
	sm.SetMemory(in, "state", stateChangeCard)
	return ""
}

func bootStateMachine(in *dt.Msg) *dt.StateMachine {
	sm := dt.NewStateMachine(pkgName)
	sm.SetDBConn(db)
	sm.SetLogger(l)
	sm.LoadState(in)
	return sm
}

var addCard []dt.State = []dt.State{
	{
		OnEntry: func(in *dt.Msg) string {
			return "Sure. You can add your card securely here: https://avabot.co/?/cards/new"
		}, OnInput: func(in *dt.Msg) {
		},
		Complete: func(in *dt.Msg) (bool, string) {
			return true, ""
		},
	},
}

var changeCard []dt.State = []dt.State{
	{
		OnEntry: func(in *dt.Msg) string {
			return "Sure. You can change your cards securely here: https://avabot.co/?/profile"
		}, OnInput: func(in *dt.Msg) {
		},
		Complete: func(in *dt.Msg) (bool, string) {
			return true, ""
		},
	},
}
