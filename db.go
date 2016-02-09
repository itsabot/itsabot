package main

import (
	"database/sql"
	"errors"

	"github.com/avabot/ava/shared/datatypes"
)

type flexIDType int

const (
	fidtInvalid flexIDType = iota // 0
	fidtEmail                     // 1
	fidtPhone                     // 2
)

var (
	ErrMissingFlexID     = errors.New("missing flexid")
	ErrInvalidFlexIDType = errors.New("invalid flexid type")
)

func saveTrainingSentence(msg *dt.Msg) (int, error) {
	q := `INSERT INTO trainings (sentence) VALUES ($1) RETURNING id`
	var id int
	if err := db.QueryRowx(q, msg.Sentence).Scan(&id); err != nil {
		return 0, err
	}
	q = `UPDATE messages SET trainingid=$1 WHERE id=$2`
	_, err := db.Exec(q, id, msg.ID)
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

// getUser returns a dt.User object based on the uid. if uid = 0, it uses the
// flexid and flexidtype to find it first.
func getUser(uid uint64, fid string, fidT flexIDType) (*dt.User, error) {
	if uid == 0 {
		fidT = fidtPhone // XXX temporary. we only have phone numbers atm
		if fid == "" {
			return nil, ErrMissingFlexID
		} else if fidT == fidtInvalid {
			return nil, ErrInvalidFlexIDType
		}
		q := `SELECT userid
		      FROM userflexids
		      WHERE flexid=$1 AND flexidtype=$2
		      ORDER BY createdat DESC`
		if err := db.Get(&uid, q, fid, fidT); err != nil {
			if err == sql.ErrNoRows {
				return nil, dt.ErrMissingUser
			}
			return nil, err
		}
	}
	q := `SELECT id, name, email, lastauthenticated, stripecustomerid
	      FROM users
	      WHERE id=$1`
	var u dt.User
	if err := db.Get(&u, q, uid); err != nil {
		// XXX if err == sql.ErrNoRows, if that also a ErrMissingUser case?
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
