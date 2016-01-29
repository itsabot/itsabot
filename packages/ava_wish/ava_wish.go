package main

import (
	"flag"
	"log"
	"math/rand"
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/nlp"
	"github.com/avabot/ava/shared/pkg"
)

var db *sqlx.DB
var p *pkg.Pkg

type Wish string

func main() {
	var coreaddr string
	flag.StringVar(&coreaddr, "coreaddr", "",
		"Port used to communicate with Ava.")
	flag.Parse()

	trigger := &nlp.StructuredInput{
		Commands: []string{"wish"},
	}

	db = connectDB()
	var err error
	p, err = pkg.NewPackage("ava_wish", coreaddr, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	wish := new(Wish)
	if err := p.Register(wish); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (pt *Wish) Run(m *dt.Msg, respMsg *dt.RespMsg) error {
	q := `INSERT INTO wishes (userid, sentence) VALUES ($1, $2)`
	_, err := db.Exec(q, m.User.ID, m.Sentence)
	if err != nil {
		return err
	}
	n := rand.Intn(5)
	switch n {
	case 0:
		m.Sentence = "Your wish is my command!"
	case 1:
		m.Sentence = "I'll make some calls."
	case 2:
		m.Sentence = "I hope to start doing that soon, too."
	case 3:
		m.Sentence = "Roger that!"
	case 4:
		m.Sentence = "I wish I could do that now, too. Soon, I hope."
	}
	return p.SaveMsg(respMsg, m)
}

func (pt *Wish) FollowUp(m *dt.Msg, respMsg *dt.RespMsg) error {
	m = dt.NewMsg(db, nil, m.User, "")
	return p.SaveMsg(respMsg, m)
}

func connectDB() *sqlx.DB {
	log.Println("connecting to db")
	var db *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=postgres dbname=ava sslmode=disable")
	}
	if err != nil {
		log.Println("err: could not connect to db", err)
	}
	return db
}
