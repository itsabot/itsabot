package dt

import "time"

type Contact struct {
	ID        uint64
	Name      string
	Email     *string
	Phone     *string
	UserID    uint64
	CreatedAt *time.Time
	UpdatedAt *time.Time
}
