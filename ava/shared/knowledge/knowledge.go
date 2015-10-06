// Package knowledge provides known and commonly required information about the
// user to 3rd party apps, such as a user's last known location.
package knowledge

import (
	"database/sql"
	"errors"

	"github.com/avabot/ava/shared/datatypes"
)

var ErrNoLocation = errors.New("no previous location")

// LastLocation returns the last known location of a user. If the location is
// unknown, LastLocation returns an error.
func LastLocation(u *datatypes.User) (*datatypes.Location, error) {
	q := `
		SELECT name, lat, lon
		FROM locations
		WHERE userid=$1
		ORDER BY createdat DESC`
	var loc *datatypes.Location
	if err := db.Get(loc, q, u.Id); err == sql.ErrNoRows {
		return loc, ErrNoLocation
	}
	return loc, err
}
