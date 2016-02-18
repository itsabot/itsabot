package main

import (
	"flag"
	"log"
	"math/rand"
	"os"

	"itsabot.org/abot/shared/datatypes"
	"itsabot.org/abot/shared/nlp"
	"itsabot.org/abot/shared/pkg"
	"github.com/jmoiron/sqlx"
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
	p, err = pkg.NewPackage("wish", coreaddr, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	wish := new(Wish)
	if err := p.Register(wish); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (pt *Wish) Run(m *dt.Msg, resp *string) error {
	q := `INSERT INTO wishes (userid, sentence) VALUES ($1, $2)`
	_, err := db.Exec(q, m.User.ID, m.Sentence)
	if err != nil {
		return err
	}
	n := rand.Intn(5)
	switch n {
	case 0:
		*resp = "Your wish is my command!"
	case 1:
		*resp = "I'll make some calls."
	case 2:
		*resp = "I hope to start doing that soon, too."
	case 3:
		*resp = "Roger that!"
	case 4:
		*resp = "I wish I could do that now, too. Soon, I hope."
	}
	return nil
}

func (pt *Wish) FollowUp(m *dt.Msg, resp *string) error {
	m = dt.NewMsg(db, nil, m.User, "")
	return nil
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
