package main

import (
	"database/sql"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
)

func saveStructuredInput(si *datatypes.StructuredInput, rsp, pkg, route string) error {
	q := `
		INSERT INTO inputs (
			userid,
			flexid,
			flexidtype,
			sentence,
			commands,
			objects,
			actors,
			times,
			places,
			response,
			package,
			route
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := db.Exec(
		q, si.UserId, si.FlexId, si.FlexIdType, si.Sentence,
		si.Commands, si.Objects, si.Actors, si.Times, si.Places, rsp,
		pkg, route)
	return err
}

func getUser(si *datatypes.StructuredInput) (*datatypes.User, error) {
	if len(si.FlexId) > 0 && si.FlexIdType == 0 {
		return nil, ErrMissingFlexIdType
	}
	var u *datatypes.User
	userQuery := `
			SELECT id, email, phone, lastauthenticated
			FROM users
			WHERE id=$1`
	if si.UserId > 0 {
		if err := db.Get(u, userQuery, si.UserId); err != nil {
			return nil, err
		}
	} else {
		q := `
			SELECT userid
			FROM usersflexids
			WHERE flexid=$1 AND flexidtype=$2
			ORDER BY createdat DESC`
		var userid int
		err := db.Get(&userid, q, si.FlexId, si.FlexIdType)
		if err != nil {
			return nil, err
		}
		if err := db.Get(u, userQuery, userid); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func getLastInput(si *datatypes.StructuredInput) (*datatypes.StructuredInput,
	error) {
	var s *datatypes.StructuredInput
	q := `SELECT (
		userid, flexid, flexidtype, commands, objects, actors,
		times, places, sentence, response, package, route) `
	if si.UserId > 0 {
		q += `WHERE userid=$1`
		if err := db.Get(s, q, si.UserId); err != nil {
			return s, err
		}
	} else {
		q += `WHERE flexid=$1 AND flexidtype=$2`
		if err := db.Get(s, q, si.FlexId, si.FlexIdType); err != nil {
			return s, err
		}
	}
	return s, nil
}

func getLastInputFromUser(u *datatypes.User) (*datatypes.StructuredInput,
	error) {
	return &(datatypes.StructuredInput{}), nil
}

func getContextObject(u *datatypes.User, si *datatypes.StructuredInput,
	datatype string) (string, error) {
	log.Debug("db: getting object context")
	var tmp *datatypes.StringSlice
	if u != nil {
		q := `
			SELECT ` + datatype + `
			FROM inputs
			WHERE userid=$1 AND array_length(objects, 1) > 0`
		if err := db.Get(&tmp, q, u.Id); err != nil {
			return "", err
		}
	} else {
		q := `
			SELECT ` + datatype + `
			FROM inputs
			WHERE (
				flexid=$1 AND
				flexidtype=$2 AND
				array_length(objects, 1) > 0)`
		err := db.Get(&tmp, q, si.FlexId, si.FlexIdType)
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
	}
	return tmp.Last(), nil
}
