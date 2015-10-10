// Package knowledge provides known and commonly required information about the
// user to 3rd party apps, such as a user's last known location.
package knowledge

import (
	"errors"

	"github.com/avabot/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
)

var ErrNoLocation = errors.New("no previous location")

// LastLocation returns the last known location of a user. If the location is
// unknown, LastLocation returns an error.
func LastLocation(db *sqlx.DB, u *datatypes.User) (*datatypes.Location, error) {
	var loc *datatypes.Location
	if u.LocationId == 0 {
		return loc, ErrNoLocation
	}
	q := `
		SELECT name, lat, lon
		FROM locations
		WHERE userid=$1
		ORDER BY createdat DESC`
	if err := db.Get(loc, q, u.Id); err != nil {
		return loc, err
	}
	return loc, nil
}
