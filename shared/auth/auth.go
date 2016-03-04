package auth

import (
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/util"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
)

const bearerAuthKey = "Bearer"

type Header struct {
	ID       uint64
	Email    string
	Scopes   []string
	IssuedAt int64
}

func LoggedIn() echo.HandlerFunc {
	return func(c *echo.Context) error {
		log.Debug("validating logged in")
		c.Response().Header().Set(echo.WWWAuthenticate, bearerAuthKey+" realm=Restricted")
		auth := c.Request().Header.Get(echo.Authorization)
		l := len(bearerAuthKey)
		// Ensure client sent the token
		if len(auth) <= l+1 || auth[:l] != bearerAuthKey {
			log.Debug("client did not send token")
			return core.JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		// Ensure the token is still valid
		tmp, err := util.CookieVal(c, "issuedAt")
		if err != nil {
			return core.JSONError(err)
		}
		issuedAt, err := strconv.ParseInt(tmp, 10, 64)
		if err != nil {
			return core.JSONError(err)
		}
		t := time.Unix(issuedAt, 0)
		if t.Add(72 * time.Hour).Before(time.Now()) {
			log.Debug("token expired")
			return core.JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		// Ensure the token has not been tampered with
		b, err := base64.StdEncoding.DecodeString(auth[l+1:])
		if err != nil {
			return core.JSONError(err)
		}
		tmp, err = util.CookieVal(c, "scopes")
		if err != nil {
			return core.JSONError(err)
		}
		scopes := strings.Fields(tmp)
		tmp, err = util.CookieVal(c, "id")
		if err != nil {
			return core.JSONError(err)
		}
		userID, err := strconv.ParseUint(tmp, 10, 64)
		if err != nil {
			return core.JSONError(err)
		}
		email, err := util.CookieVal(c, "email")
		if err != nil {
			return core.JSONError(err)
		}
		a := Header{
			ID:       userID,
			Email:    email,
			Scopes:   scopes,
			IssuedAt: issuedAt,
		}
		byt, err := json.Marshal(a)
		if err != nil {
			return core.JSONError(err)
		}
		known := hmac.New(sha512.New, []byte(os.Getenv("ABOT_SECRET")))
		_, err = known.Write(byt)
		if err != nil {
			return core.JSONError(err)
		}
		ok := hmac.Equal(known.Sum(nil), b)
		if !ok {
			log.Debug("token tampered")
			return core.JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		log.Debug("validated logged in")
		return nil
	}
}

// validateCSRF ensures that any forms posted to Abot are protected against
// Cross-Site Request Forgery. Without this function, Abot would be vulnerable
// to the attack because tokens are stored client-side in cookies.
func CSRF(db *sqlx.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		// TODO look into other session-based temporary storage systems
		// for these csrf tokens to prevent hitting the database.
		// Whatever is selected must *not* introduce a dependency
		// (memcached/Redis). Bolt might be an option.
		if c.Request().Method == "GET" {
			return nil
		}
		log.Debug("validating csrf")
		var label string
		q := `SELECT label FROM sessions
		      WHERE userid=$1 AND label='csrfToken' AND token=$2`
		uid, err := util.CookieVal(c, "id")
		if err != nil {
			return core.JSONError(err)
		}
		token, err := util.CookieVal(c, "csrfToken")
		if err != nil {
			return core.JSONError(err)
		}
		err = db.Get(&label, q, uid, token)
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		if err != nil {
			return core.JSONError(err)
		}
		log.Debug("validated csrf")
		return nil
	}
}

// Admin ensures that the current user is an admin. We trust the scopes
// presented by the client because they're validated through HMAC in LoggedIn().
func Admin() echo.HandlerFunc {
	return func(c *echo.Context) error {
		log.Debug("validating admin")
		tmp, err := util.CookieVal(c, "scopes")
		if err != nil {
			return core.JSONError(err)
		}
		scopes := strings.Fields(tmp)
		for _, scope := range scopes {
			if scope == "admin" {
				log.Debug("validated admin")
				return nil
			}
		}
		return core.JSONError(echo.NewHTTPError(http.StatusUnauthorized))
	}
}
