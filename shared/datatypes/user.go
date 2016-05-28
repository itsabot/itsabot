package dt

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/itsabot/abot/core/log"
	"github.com/jmoiron/sqlx"
)

// User represents a user, which is usually the user that sent a message to
// Abot.
type User struct {
	Name     string
	Email    string
	Password string // temp storage prior to hashing
	ID       uint64
	Admin    bool

	// FlexID and FlexIDType are particularly useful when a user has not
	// yet registered.
	FlexID     string
	FlexIDType FlexIDType

	// Trainer determines whether the user has access to the training
	// interface and will be notified via email when new training is
	// required
	Trainer bool
}

// FlexIDType is used to identify a user when only an email, phone, or other
// "flexible" ID is available.
type FlexIDType int

// FlexIDTypes are named enum values for the various methods of communicating
// with Abot.
const (
	FIDTEmail FlexIDType = iota + 1 // 1
	FIDTPhone                       // 2

	// FIDTSession is used to track a user's session where no other
	// information like email or phone is obtained, e.g. communicating to
	// Abot via a website.
	FIDTSession // 3
)

// ErrMissingFlexIDType is returned when a FlexIDType is expected, but
// none found.
var ErrMissingFlexIDType = errors.New("missing flexidtype")

// ErrMissingFlexID is returned when a FlexID is expected, but none
// found.
var ErrMissingFlexID = errors.New("missing flexid")

// ErrInvalidFlexIDType is returned when a FlexIDType is invalid not
// matching one of the pre-defined FlexIDTypes for email (1) or phone
// (2).
var ErrInvalidFlexIDType = errors.New("invalid flexid type")

// GetUser from an HTTP request.
func GetUser(db *sqlx.DB, req *Request) (*User, error) {
	u := &User{}
	u.FlexID = req.FlexID
	u.FlexIDType = req.FlexIDType
	if req.UserID == 0 {
		if req.FlexID == "" {
			return nil, ErrMissingFlexID
		}
		switch req.FlexIDType {
		case FIDTEmail, FIDTPhone, FIDTSession:
			// Do nothing
		default:
			return nil, ErrInvalidFlexIDType
		}
		log.Debug("searching for user from", req.FlexID, req.FlexIDType)
		q := `SELECT userid
		      FROM userflexids
		      WHERE flexid=$1 AND flexidtype=$2
		      ORDER BY createdat DESC`
		err := db.Get(&req.UserID, q, req.FlexID, req.FlexIDType)
		if err == sql.ErrNoRows {
			return u, nil
		}
		log.Debug("got uid", req.UserID)
		if err != nil {
			return nil, err
		}
	}
	q := `SELECT id, name, email FROM users WHERE id=$1`
	if err := db.Get(u, q, req.UserID); err != nil {
		if err == sql.ErrNoRows {
			return u, nil
		}
		return nil, err
	}
	return u, nil
}

// Create a new user in the database.
func (u *User) Create(db *sqlx.DB, fidT FlexIDType, fid string) error {
	// Create the password hash
	hpw, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
	if err != nil {
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	q := `INSERT INTO users (name, email, password, locationid, admin)
	      VALUES ($1, $2, $3, 0, $4)
	      RETURNING id`
	var uid uint64
	err = tx.QueryRowx(q, u.Name, u.Email, hpw, u.Admin).Scan(&uid)
	if err != nil && err.Error() ==
		`pq: duplicate key value violates unique constraint "users_email_key"` {
		_ = tx.Rollback()
		return err
	}
	if uid == 0 {
		_ = tx.Rollback()
		return err
	}
	q = `INSERT INTO userflexids (userid, flexid, flexidtype)
	     VALUES ($1, $2, $3)`
	_, err = tx.Exec(q, uid, u.Email, 1)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	_, err = tx.Exec(q, uid, fid, 2)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	u.ID = uid
	return nil
}

// DeleteSessions removes any open sessions by the user. This enables "logging
// out" of the web-based client.
func (u *User) DeleteSessions(db *sqlx.DB) error {
	q := `DELETE FROM sessions WHERE userid=$1`
	_, err := db.Exec(q, u.ID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}
