package dt

import (
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/itsabot/abot/core/log"
	"github.com/jmoiron/sqlx"
)

// User represents a user, which is usually the user that sent a message to
// Abot.
type User struct {
	Name                     string
	Email                    string
	Password                 string // temp storage prior to hashing
	PaymentServiceID         string
	LocationID               int
	ID                       uint64
	AuthorizationID          sql.NullInt64
	LastAuthenticationMethod AuthMethod
	LastAuthenticated        *time.Time
	Admin                    bool

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

const (
	fidtEmail FlexIDType = iota + 1 // 1
	fidtPhone                       // 2

	// fidtSession is used to track a user's session where no other
	// information like email or phone is obtained, e.g. communicating to
	// Abot via a website.
	fidtSession // 3
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
		case fidtEmail, fidtPhone, fidtSession:
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
	q := `SELECT id, name, email, lastauthenticated, paymentserviceid
	      FROM users
	      WHERE id=$1`
	if err := db.Get(u, q, req.UserID); err != nil {
		if err == sql.ErrNoRows {
			return u, nil
		}
		return nil, err
	}
	return u, nil
}

// GetName satisfies the Contactable interface
func (u *User) GetName() string {
	return u.Name
}

// GetEmail satisfies the Contactable interface
func (u *User) GetEmail() string {
	return u.Email
}

// Registered checks whether the current user has signed up or associated his
// flexID with this user account.
func (u *User) Registered() bool {
	return u.ID > 0
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
	q := `INSERT INTO users (name, email, password, locationid)
	      VALUES ($1, $2, $3, 0)
	      RETURNING id`
	var uid uint64
	err = tx.QueryRowx(q, u.Name, u.Email, hpw).Scan(&uid)
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

// IsAuthenticated confirms that the user is authenticated for a particular
// AuthMethod.
func (u *User) IsAuthenticated(m AuthMethod) (bool, error) {
	var oldTime time.Time
	tmp := os.Getenv("ABOT_REQUIRE_AUTH_IN_HOURS")
	var t int
	if len(tmp) > 0 {
		var err error
		t, err = strconv.Atoi(tmp)
		if err != nil {
			return false, err
		}
		if t < 0 {
			return false, errors.New("negative ABOT_REQUIRE_AUTH_IN_HOURS")
		}
	} else {
		log.Debug("ABOT_REQUIRE_AUTH_IN_HOURS environment variable is not set.",
			" Using 168 hours (one week) as the default.")
		t = 168
	}
	oldTime = time.Now().Add(time.Duration(-1*t) * time.Hour)
	authenticated := false
	if u.LastAuthenticated.After(oldTime) &&
		u.LastAuthenticationMethod >= m {
		authenticated = true
	}
	return authenticated, nil
}

// GetCards retrieves credit cards for a specific user.
func (u *User) GetCards(db *sqlx.DB) ([]Card, error) {
	q := `SELECT id, addressid, last4, cardholdername, expmonth, expyear,
	          brand, stripeid, zip5hash
	      FROM cards
	      WHERE userid=$1`
	var cards []Card
	err := db.Select(&cards, q, u.ID)
	return cards, err
}

// GetPrimaryCard retrieves the primary credit card for a specific user.
func (u *User) GetPrimaryCard(db *sqlx.DB) (*Card, error) {
	q := `SELECT id, addressid, last4, cardholdername, expmonth, expyear,
	          brand, stripeid
	      FROM cards
	      WHERE userid=$1 AND primary=TRUE`
	var card *Card
	if err := db.Get(&card, q, u.ID); err != nil {
		return card, err
	}
	return card, nil
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

// SaveAddress of a specific user to the database.
func (u *User) SaveAddress(db *sqlx.DB, addr *Address) (uint64, error) {
	q := `INSERT INTO addresses
	      (userid, cardid, name, line1, line2, city, state, country, zip,
	          zip5, zip4)
	      VALUES ($1, 0, $2, $3, $4, $5, $6, 'USA', $7, $8, $9)
	      RETURNING id`
	var id uint64
	err := db.QueryRowx(q, u.ID, addr.Name, addr.Line1, addr.Line2,
		addr.City, addr.State, addr.Zip, addr.Zip5, addr.Zip4).Scan(&id)
	return id, err
}

// GetAddress standardizes the name of addresses for faster searching and
// consistent responses.
func (u *User) GetAddress(db *sqlx.DB, text string) (*Address, error) {
	addr := &Address{}
	var name string
	for _, w := range strings.Fields(strings.ToLower(text)) {
		switch w {
		case "home", "place", "apartment", "flat", "house", "condo":
			name = "home"
		case "work", "office", "biz", "business":
			name = "office"
		}
	}
	if len(name) == 0 {
		log.Debug("no address found: " + text)
		return nil, ErrNoAddress
	}
	q := `SELECT name, line1, line2, city, state, country, zip
	      FROM addresses
	      WHERE userid=$1 AND name=$2 AND cardid=0`
	err := db.Get(addr, q, u.ID, name)
	if err == sql.ErrNoRows {
		return nil, ErrNoAddress
	}
	if err != nil {
		log.Debug("GET returned no address for", name)
		return nil, err
	}
	return addr, nil
}

// UpdateAddressName such as "home" or "office" when learned.
func (u *User) UpdateAddressName(db *sqlx.DB, id uint64, name string) (*Address,
	error) {
	q := `UPDATE addresses SET name=$1 WHERE id=$2`
	if _, err := db.Exec(q, name, id); err != nil {
		return nil, err
	}
	q = `SELECT name, line1, line2, city, state, country, zip
	     FROM addresses
	     WHERE id=$1`
	addr := &Address{}
	if err := db.Get(addr, q, id); err != nil {
		return nil, err
	}
	return addr, nil
}

// CheckActiveAuthorization determines if a message to Ava was fulfilling an
// authorization request. RequestAuth nulls out the authorizationid once auth
// has been completed.
func (u *User) CheckActiveAuthorization(db *sqlx.DB) (bool, error) {
	q := `SELECT authorizationid FROM users WHERE id=$1`
	var authID sql.NullInt64
	if err := db.Get(&authID, q, u.ID); err != nil {
		return false, err
	}
	if !authID.Valid {
		return false, nil
	}
	return true, nil
}

// AuthMethod allows you as the plugin developer to control the level of
// security required in an authentication. Select an appropriate security level
// depending upon your risk tolerance for fraud compared against the quality and
// ease of the user experience.
//
// NOTE this is just a stub and isn't implemented
// TODO build the constants defining the types of AuthMethods
type AuthMethod int
