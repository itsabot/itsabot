package main

import (
	"database/sql"
	"errors"
	"log"

	"github.com/avabot/ava/shared/datatypes"
)

var ErrMissingUser = errors.New("missing user")

func saveStructuredInput(in *datatypes.Input, rsp, pkg,
	route string) error {
	// TODO
	q := `
		INSERT INTO responses (userid, inputid, response, state)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	// TODO Change 0 to $10 below with responseid
	q = `
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
			responseid,
			package,
			route
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, $10, $11)`
	si := in.StructuredInput
	_, err := db.Exec(
		q, in.UserId, in.FlexId, in.FlexIdType, in.Sentence,
		si.Commands, si.Objects, si.Actors, si.Times, si.Places,
		pkg, route)
	return err
}

func getUser(in *datatypes.Input) (*datatypes.User, error) {
	if len(in.FlexId) > 0 && in.FlexIdType == 0 {
		return nil, ErrMissingFlexIdType
	}
	if in.UserId == 0 {
		q := `SELECT userid
		      FROM userflexids
		      WHERE flexid=$1 AND flexidtype=$2
		      ORDER BY createdat DESC`
		err := db.Get(&in.UserId, q, in.FlexId, in.FlexIdType)
		if err == sql.ErrNoRows {
			return nil, ErrMissingUser
		} else if err != nil {
			return nil, err
		}
	}
	q := `SELECT id, name, email, lastauthenticated
	      FROM users
	      WHERE id=$1`
	var u *datatypes.User
	if err := db.Get(u, q, in.UserId); err != nil {
		return nil, err
	}
	return u, nil
}

func getLastInput(in *datatypes.Input) (*datatypes.Input, error) {
	var input *datatypes.Input
	q := `SELECT (
		userid, flexid, flexidtype, commands, objects, actors,
		times, places, sentence, responseid, package, route) `
	if in.UserId > 0 {
		q += `WHERE userid=$1`
		if err := db.Get(input, q, in.UserId); err != nil {
			log.Println("err: ", err)
			return input, err
		}
	}
	return input, nil
}

func getLastInputFromUser(u *datatypes.User) (*datatypes.StructuredInput,
	error) {
	return &(datatypes.StructuredInput{}), nil
}

func getContextObject(u *datatypes.User, si *datatypes.StructuredInput,
	datatype string) (string, error) {
	log.Println("db: getting object context")
	var tmp *datatypes.StringSlice
	if u == nil {
		return "", ErrMissingUser
	}
	if u != nil {
		q := `
			SELECT ` + datatype + `
			FROM inputs
			WHERE userid=$1 AND array_length(objects, 1) > 0`
		if err := db.Get(&tmp, q, u.Id); err != nil {
			return "", err
		}
	}
	return tmp.Last(), nil
}
