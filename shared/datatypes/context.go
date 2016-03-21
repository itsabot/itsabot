package dt

import (
	"github.com/itsabot/abot/core/log"
	"github.com/jmoiron/sqlx"
)

type Context struct {
	DB  *sqlx.DB
	Log *log.Logger
}
