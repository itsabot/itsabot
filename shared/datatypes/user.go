package dt

import (
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itsabot/abot/shared/log"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
)

type User struct {
	ID                       uint64
	Name                     string
	Email                    string
	LocationID               int
	StripeCustomerID         string
	AuthorizationID          sql.NullInt64
	LastAuthenticated        *time.Time
	LastAuthenticationMethod AuthMethod

	// Trainer determines whether the user has access to the training
	// interface and will be notified via email when new training is
	// required
	Trainer bool
}

// AuthMethod allows you as the package developer to control the level of
// security required in an authentication. Select an appropriate security level
// depending upon your risk tolerance for fraud compared against the quality and
// ease of the user experience.
//
// NOTE this is just a stub and isn't implemented
// TODO build the constants defining the types of AuthMethods
type AuthMethod int

// FlexIDType is used to identify a user when only an email, phone, or other
// "flexible" ID is available.
type FlexIDType int

// userParams holds the identifiers for a user used in a message.
type userParams struct {
	UserID uint64
	FlexID string
	FlexIDType
}

const (
	fidtInvalid FlexIDType = iota // 0
	fidtEmail                     // 1
	fidtPhone                     // 2
)

var (
	ErrMissingUser       = errors.New("missing user")
	ErrMissingFlexIdType = errors.New("missing flexidtype")
	ErrMissingFlexID     = errors.New("missing flexid")
	ErrInvalidFlexIDType = errors.New("invalid flexid type")
)

func GetUser(db *sqlx.DB, c *echo.Context) (*User, error) {
	p, err := extractUserParams(db, c)
	if err != nil {
		return nil, err
	}
	log.Debug("extracted user params", p)
	if p.UserID == 0 {
		// XXX temporary. we only have phone numbers atm
		p.FlexIDType = fidtPhone
		if p.FlexID == "" {
			return nil, ErrMissingFlexID
		} else if p.FlexIDType == fidtInvalid {
			return nil, ErrInvalidFlexIDType
		}
		log.Debug("searching for user from", p.FlexID, p.FlexIDType)
		q := `SELECT userid
		      FROM userflexids
		      WHERE flexid=$1 AND flexidtype=$2
		      ORDER BY createdat DESC`
		err := db.Get(&p.UserID, q, p.FlexID, p.FlexIDType)
		if err == sql.ErrNoRows {
			return nil, ErrMissingUser
		}
		log.Debug("got uid", p.UserID)
		if err != nil {
			return nil, err
		}
	}
	q := `SELECT id, name, email, lastauthenticated, stripecustomerid
	      FROM users
	      WHERE id=$1`
	u := &User{}
	if err := db.Get(u, q, p.UserID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMissingUser
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

func (u *User) GetCards(db *sqlx.DB) ([]Card, error) {
	q := `
		SELECT id, addressid, last4, cardholdername, expmonth, expyear,
		       brand, stripeid, zip5hash
		FROM cards
		WHERE userid=$1`
	log.Debug("getting cards for user", u.ID)
	var cards []Card
	err := db.Select(&cards, q, u.ID)
	return cards, err
}

func (u *User) GetPrimaryCard(db *sqlx.DB) (*Card, error) {
	q := `
		SELECT id, addressid, last4, cardholdername, expmonth, expyear,
		       brand, stripeid
		FROM cards
		WHERE userid=$1 AND primary=TRUE`
	var card *Card
	if err := db.Get(&card, q, u.ID); err != nil {
		return card, err
	}
	return card, nil
}

func (u *User) DeleteSessions(db *sqlx.DB) error {
	q := `DELETE FROM sessions WHERE userid=$1`
	_, err := db.Exec(q, u.ID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

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
	q := `
		SELECT name, line1, line2, city, state, country, zip
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

// extractUserParams splits out user-identifying params passed in through
// endpoints.
func extractUserParams(db *sqlx.DB, c *echo.Context) (*userParams, error) {
	p := &userParams{}
	tmp, ok := c.Get("uid").(string)
	if !ok {
		tmp = ""
	}
	var err error
	if len(tmp) > 0 {
		p.UserID, err = strconv.ParseUint(tmp, 10, 64)
		if err != nil && err.Error() != `strconv.ParseInt: parsing "": invalid syntax` {
			return p, err
		}
	}
	if p.UserID > 0 {
		return p, nil
	}
	tmp, ok = c.Get("flexid").(string)
	if !ok {
		tmp = ""
	}
	if len(tmp) == 0 {
		return p, errors.New("flexid is blank")
	}
	p.FlexID = tmp
	tmp, ok = c.Get("flexidtype").(string)
	if !ok {
		tmp = ""
	}
	var typ int
	if len(tmp) > 0 {
		typ, err = strconv.Atoi(tmp)
		if err != nil && err.Error() ==
			`strconv.ParseInt: parsing "": invalid syntax` {
			// default to 2 (SMS)
			p.FlexIDType = FlexIDType(2)
		} else if err != nil {
			return p, err
		}
	}
	p.FlexIDType = FlexIDType(typ)
	return p, nil
}
