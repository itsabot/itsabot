package dt

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
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

const (
	FlexIdTypeEmail int = iota + 1
	FlexIdTypePhone
)

var ErrNoAddress = errors.New("no address")

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
	tmp := os.Getenv("REQUIRE_AUTH_IN_HOURS")
	var t int
	if len(tmp) > 0 {
		var err error
		t, err = strconv.Atoi(tmp)
		if err != nil {
			return false, err
		}
		if t < 0 {
			return false, errors.New("negative REQUIRE_AUTH_IN_HOURS")
		}
	} else {
		log.Println("REQUIRE_AUTH_IN_HOURS environment variable is not set.",
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
	log.Println("getting cards for user", u.ID)
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
		log.Println("no address found: " + text)
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
		log.Println("GET returned no address for", name)
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
	q = `
		SELECT name, line1, line2, city, state, country, zip
		FROM addresses
		WHERE id=$1`
	addr := &Address{}
	if err := db.Get(addr, q, id); err != nil {
		return nil, err
	}
	return addr, nil
}
