package prefs

import (
	"database/sql"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

const (
	KeyBudget string = "budget"
	KeyTaste         = "taste"
)

type Global bool

func Get(db *sqlx.DB, uid uint64, pkgName, k string) (string, error) {
	var q string
	if len(pkgName) == 0 {
		q = `
			SELECT value
			FROM preferences
			WHERE key=$1 AND userid=$2 AND pkgname=NULL
			ORDER BY createdat DESC`
	} else {
		q = `
			SELECT value
			FROM preferences
			WHERE key=$1 AND userid=$2 AND pkgname=$3
			ORDER BY createdat DESC`
	}
	var v string
	err := db.Get(&v, q, k, uid, pkgName)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	return v, nil
}

func Save(db *sqlx.DB, uid uint64, pkgName, k, v string) error {
	q := `
		INSERT INTO preferences
		(key, value, pkgname, userid) VALUES ($1, $2, $3, $4)`
	if _, err := db.Exec(q, k, v, pkgName, uid); err != nil {
		return err
	}
	return nil
}
