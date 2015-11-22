package datatypes

type Phone struct {
	Id     int
	Number string `db:"flexid"`
}
