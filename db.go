package main

import (
	"database/sql"
	"errors"

	"github.com/avabot/ava/shared/datatypes"
)

var (
	ErrMissingFlexID = errors.New("missing flexid")
)

func saveMsg(m *dt.Msg) error {
	q := `INSERT INTO messages (
		  userid,
		  sentence,
		  sentenceannotated,
		  package,
		  route
		) VALUES ($1, $2, $3, $4, $5)`
	si := m.StructuredInput
	_, err := db.Exec(q, m.User.ID, m.Sentence, m.SentenceAnnotated,
		si.Commands, si.Objects, si.Actors, si.Times, si.Places,
		m.Package, m.Route)
	return err
}

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

func getUser(uid uint64, fid string, fidT int) (*dt.User, error) {
	if uid == 0 {
		q := `SELECT userid
		      FROM userflexids
		      WHERE flexid=$1 AND flexidtype=2
		      ORDER BY createdat DESC`
		err := db.Get(&uid, q, fid)
		if err == sql.ErrNoRows {
			return nil, dt.ErrMissingUser
		} else if err != nil {
			return nil, err
		}
	} else if len(fid) == 0 {
		return nil, ErrMissingFlexID
	}
	q := `SELECT id, name, email, lastauthenticated, stripecustomerid
	      FROM users
	      WHERE id=$1`
	u := dt.User{}
	if err := db.Get(&u, q, uid); err != nil {
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
