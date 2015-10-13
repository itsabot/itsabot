package datatypes

import (
	"errors"
	"os"
	"strconv"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

type User struct {
	Id                int
	Name              string
	Email             string
	LocationId        int
	LastAuthenticated *time.Time
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
		log.Warn("REQUIRE_AUTH_IN_HOURS environment variable is not set.",
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
