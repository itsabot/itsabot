package dt

import (
	"os"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/shared/sms"
)

type Ctx struct {
	Msg *Msg
	DB  *sqlx.DB
	SG  *MailClient
	EC  *SearchClient
	TC  *twilio.Client
}

func NewContext() (*Ctx, error) {
	db, err := NewDatabaseConn()
	if err != nil {
		return nil, err
	}
	return &Ctx{
		DB: db,
		SG: NewMailClient(),
		EC: NewSearchClient(),
		TC: sms.NewClient(),
	}, nil
}

func NewDatabaseConn() (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		db, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		db, err = sqlx.Connect("postgres",
			"user=egtann dbname=ava sslmode=disable")
	}
	return db, err
}
