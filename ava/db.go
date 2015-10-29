package main

import (
	"database/sql"
	"errors"
	"log"

	"github.com/avabot/ava/shared/datatypes"
)

var (
	ErrMissingUser       = errors.New("missing user")
	ErrMissingFlexIDType = errors.New("missing flexidtype")
)

func saveStructuredInput(m *datatypes.Message, rid int, pkg,
	route string) error {
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
			responseid,
			package,
			route
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	in := m.Input
	si := in.StructuredInput
	_, err := db.Exec(
		q, in.UserID, in.FlexID, in.FlexIDType, in.Sentence,
		si.Commands, si.Objects, si.Actors, si.Times, si.Places, rid,
		pkg, route)
	return err
}

func saveTrainingSentence(s string) (int, error) {
	q := `INSERT INTO trainings (sentence) VALUES ($1) RETURNING id`
	var id int
	row := db.QueryRowx(q, s)
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func getUser(in *datatypes.Input) (*datatypes.User, error) {
	if in.UserID == 0 {
		q := `SELECT userid
		      FROM userflexids
		      WHERE flexid=$1 AND flexidtype=$2
		      ORDER BY createdat DESC`
		err := db.Get(&in.UserID, q, in.FlexID, in.FlexIDType)
		if err == sql.ErrNoRows {
			return nil, ErrMissingUser
		} else if err != nil {
			return nil, err
		}
	} else if len(in.FlexID) > 0 && in.FlexIDType == 0 {
		return nil, ErrMissingFlexIDType
	}
	q := `SELECT id, name, email, lastauthenticated
	      FROM users
	      WHERE id=$1`
	u := datatypes.User{}
	if err := db.Get(&u, q, in.UserID); err != nil {
		return nil, err
	}
	return &u, nil
}

func getLastInput(in *datatypes.Input) (*datatypes.Input, error) {
	var input *datatypes.Input
	q := `SELECT (
		userid, flexid, flexidtype, commands, objects, actors,
		times, places, sentence, responseid, package, route) `
	if in.UserID > 0 {
		q += `WHERE userid=$1`
		if err := db.Get(input, q, in.UserID); err != nil {
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
		if err := db.Get(&tmp, q, u.ID); err != nil {
			return "", err
		}
	}
	return tmp.Last(), nil
}
