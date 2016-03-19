package dt

import (
	"github.com/itsabot/abot/shared/log"
	"github.com/jmoiron/sqlx"
)

type Context struct {
	DB  *sqlx.DB
	Log *log.Logger
}
