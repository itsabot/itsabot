package datatypes

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

type User struct {
	ID                       int
	Name                     string
	Email                    string
	LocationID               int
	StripeCustomerID         string
	AuthorizationID          sql.NullInt64
	LastAuthenticated        *time.Time
	LastAuthenticationMethod Method
}

func (u *User) IsAuthenticated(m Method) (bool, error) {
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
		       brand, stripeid
		FROM cards
		WHERE userid=$1`
	var cards []Card
	rows, err := db.Queryx(q, u.ID)
	if err != nil {
		return cards, err
	}
	var card Card
	for rows.Next() {
		if err = rows.StructScan(&card); err != nil {
			return cards, err
		}
		cards = append(cards, card)
	}
	return cards, nil
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

func (u *User) SaveAddress(db *sqlx.DB, addr *Address) error {
	q := `
		INSERT INTO addresses
		(userid, cardid, name, line1, line2, city, state, country, zip)
		VALUES ($1, 0, $2, $3, $4, $5, $6, $7, $8)`
	_, err := db.Exec(q, u.ID, addr.Name, addr.Line1, addr.Line2,
		addr.City, addr.State, addr.Country, addr.Zip)
	return err
}
