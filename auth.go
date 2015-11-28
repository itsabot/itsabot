package main

import (
	"database/sql"

	"github.com/avabot/ava/shared/datatypes"
)

// checkActiveAuthorization determines if a message to Ava was fulfilling an
// authorization request. RequestAuth nulls out the authorizationid once auth
// has been completed.
func checkActiveAuthorization(m *dt.Msg) (bool, error) {
	q := `SELECT authorizationid FROM users WHERE id=$1`
	var authID sql.NullInt64
	if err := db.Get(&authID, q, m.User.ID); err != nil {
		return false, err
	}
	if !authID.Valid {
		return false, nil
	}
	return true, nil
}
