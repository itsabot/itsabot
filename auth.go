package main

import "github.com/avabot/ava/shared/datatypes"

// checkActiveAuthorization determines if a message to Ava was fulfilling an
// authorization request.
func checkActiveAuthorization(m *dt.Msg) (bool, error) {
	q := `
		SELECT COUNT(id) FROM authorizations
		WHERE userid=$1 AND attempts=0 AND authorizedat=NULL`
	var count uint64
	if err := db.Select(&count, q, m.User.ID); err != nil {
		return false, err
	}
	return count > 0, nil
}
