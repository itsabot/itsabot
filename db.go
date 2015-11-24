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

func saveStructuredInput(m *dt.Msg, rid int, pkg,
	route string) (int, error) {
	q := `
		INSERT INTO inputs (
			userid,
			flexid,
			flexidtype,
			sentence,
			sentenceannotated,
			commands,
			objects,
			actors,
			times,
			places,
			responseid,
			package,
			route
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`
	in := m.Input
	si := in.StructuredInput
	row := db.QueryRowx(
		q, in.UserID, in.FlexID, in.FlexIDType, in.Sentence, in.SentenceAnnotated,
		si.Commands, si.Objects, si.Actors, si.Times, si.Places, rid,
		pkg, route)
	var id int
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func saveTrainingSentence(in *dt.Input) (int, error) {
	q := `INSERT INTO trainings (sentence) VALUES ($1) RETURNING id`
	var id int
	row := db.QueryRowx(q, in.Sentence)
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	q = `UPDATE inputs SET trainingid=$1 WHERE id=$2`
	log.Println("updating input", id, in.ID)
	_, err := db.Exec(q, id, in.ID)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func updateTraining(trainID int, hitID string, maxAssignments uint) error {
	q := `UPDATE trainings SET foreignid=$1, maxassignments=$2 WHERE id=$3`
	_, err := db.Exec(q, hitID, maxAssignments, trainID)
	if err != nil {
		return err
	}
	return nil
}

func getUser(in *dt.Input) (*dt.User, error) {
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
	u := dt.User{}
	if err := db.Get(&u, q, in.UserID); err != nil {
		return nil, err
	}
	return &u, nil
}

func getInputAnnotation(id int) (string, error) {
	var annotation string
	q := `SELECT sentenceannotated FROM inputs WHERE trainingid=$1`
	if err := db.Get(&annotation, q, id); err != nil {
		return "", err
	}
	return annotation, nil
}

func getLastInputFromUser(u *dt.User) (*dt.StructuredInput,
	error) {
	return &(dt.StructuredInput{}), nil
}

func getContextObject(u *dt.User, si *dt.StructuredInput,
	datatype string) (string, error) {
	log.Println("db: getting object context")
	var tmp *dt.StringSlice
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
