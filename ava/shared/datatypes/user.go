package datatypes

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"
)

type User struct {
	ID                int
	Name              string
	Email             string
	LocationID        int        `db:"locationid"`
	LastAuthenticated *time.Time `db:"lastauthenticated"`
}

func (u *User) isAuthenticated() (bool, error) {
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
	if u.LastAuthenticated.After(oldTime) {
		authenticated = true
	}
	return authenticated, nil
}
