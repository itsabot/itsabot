package dt

// Phone represents a phone as a flexid from the database.
type Phone struct {
	Id     uint64
	Number string `db:"flexid"`
}
