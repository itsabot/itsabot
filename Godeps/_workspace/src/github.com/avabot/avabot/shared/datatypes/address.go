package dt

import "github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"

type Address struct {
	ID             uint64
	Name           string
	Line1          string
	Line2          string
	City           string
	State          string
	Zip            string
	Zip5           string
	Zip4           string
	Country        string
	DisplayAddress string
}

func GetAddress(dest *Address, db *sqlx.DB, id uint64) error {
	q := `SELECT id, line1, line2, city, state, country, zip WHERE id=$1`
	if err := db.Get(dest, q, id); err != nil {
		return err
	}
	return nil
}
