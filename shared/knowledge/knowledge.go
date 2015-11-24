// Package knowledge provides known and commonly required information about the
// user to 3rd party apps, such as a user's last known location.
package knowledge

import (
	"database/sql"
	"errors"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

var ErrNoLocation = errors.New("no previous location")

// GetLocation returns the last known location of a user. If the location isn't
// recent, ask the user to confirm.
func GetLocation(db *sqlx.DB, u *dt.User) (*dt.Location, string,
	error) {
	var loc *dt.Location
	if u.LocationID == 0 {
		return loc, language.QuestionLocation(""), nil
	}
	q := `
		SELECT name, createdat
		FROM locations
		WHERE userid=$1
		ORDER BY createdat DESC`
	err := db.Get(loc, q, u.ID)
	if err == sql.ErrNoRows {
		return loc, language.QuestionLocation(""), nil
	} else if err != nil {
		return loc, "", err
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	if loc.CreatedAt.Before(yesterday) {
		return loc, language.QuestionLocation(loc.Name), nil
	}
	return loc, "", nil
}
