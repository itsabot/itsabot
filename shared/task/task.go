package task

import (
	"database/sql"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
)

type task struct {
	Done bool
	Err  error
	pkg  string
	db   *sqlx.DB
	u    *dt.User
	msg  *dt.Msg
}

func New(db *sqlx.DB, u *dt.User, pkgName string) *task {
	return &task{
		db:  db,
		u:   u,
		pkg: pkgName,
	}
}

func (t *task) RequestAddress(dest *dt.Address, resp *dt.Resp,
	respMsg *dt.RespMsg) (bool, error) {
	table := "addresses"
	q := `
		SELECT resultid
		FROM tasks
		WHERE userid=$1 AND resulttable='$2'`
	var resID sql.NullInt64
	err := t.db.Get(&resID, q, t.u.ID, table)
	if err == sql.ErrNoRows {
		if err = t.save(table); err != nil {
			// kick off the address request process
			return false, err
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !resID.Valid {
		return false, nil
	}
	if err = t.u.GetAddress(dest, resID); err != nil {
		return false, err
	}
	return true, nil
}

func (t *task) save(table string) error {
	q := `
		INSERT INTO tasks (userid, packagename, resulttable)
		VALUES ($1, $2, $3, $4)`
	_, err := t.db.Exec(q, t.u.ID, t.pkg, table)
	return err
}
