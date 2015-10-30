package datatypes

import "time"

type Location struct {
	Name      string
	Lat       float64
	Lon       float64
	CreatedAt time.Time
}

func (l Location) IsRecent() bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return l.CreatedAt.After(yesterday)
}
