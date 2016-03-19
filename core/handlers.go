package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	w "golang.org/x/net/websocket"

	"github.com/itsabot/abot/core/websocket"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/interface/emailsender"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/util"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
)

var tmplLayout *template.Template
var ws = websocket.NewAtomicWebSocketSet()

// ErrInvalidUserPass reports an invalid username/password combination during
// login.
var ErrInvalidUserPass = errors.New("Invalid username/password combination")

// initRoutes when creating a new server.
func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger(), mw.Gzip(), mw.Recover())
	e.SetDebug(true)
	logger := log.New("")
	e.SetLogger(logger)

	e.Static("/public/css", "public/css")
	e.Static("/public/js", "public/js")
	e.Static("/public/images", "public/images")

	if os.Getenv("ABOT_ENV") != "production" {
		cmd := e.Group("/_/cmd")
		initCMDGroup(cmd)
	}

	// Web routes
	e.Get("/*", HandlerIndex)
	e.Post("/", HandlerMain)

	// API routes (no restrictions)
	e.Post("/api/login.json", HandlerAPILoginSubmit)
	e.Post("/api/logout.json", HandlerAPILogoutSubmit)
	e.Post("/api/signup.json", HandlerAPISignupSubmit)
	e.Post("/api/forgot_password.json", HandlerAPIForgotPasswordSubmit)
	e.Post("/api/reset_password.json", HandlerAPIResetPasswordSubmit)

	// API routes (restricted by login)
	var api *echo.Group
	if os.Getenv("ABOT_ENV") == "production" {
		api = e.Group("/api/user", LoggedIn(), CSRF(db))
	} else {
		api = e.Group("/api/user", LoggedIn())
	}
	api.Get("/profile.json", HandlerAPIProfile)
	api.Put("/profile.json", HandlerAPIProfileView)

	// API routes (restricted to admins)
	var apiAdmin *echo.Group
	if os.Getenv("ABOT_ENV") == "production" {
		apiAdmin = e.Group("/api/admin", LoggedIn(), CSRF(db), Admin())
	} else {
		apiAdmin = e.Group("/api/admin", LoggedIn(), Admin())
	}
	apiAdmin.Get("/plugins.json", HandlerAPIPlugins)

	// WebSockets
	e.WebSocket("/ws", HandlerWSConversations)
}

// HandlerIndex presents the homepage to the user and populates the HTML with
// server-side variables.
func HandlerIndex(c *echo.Context) error {
	if os.Getenv("ABOT_ENV") != "development" {
		var err error
		tmplLayout, err = template.ParseFiles("assets/html/layout.html")
		if err != nil {
			return err
		}
		if err = CompileAssets(); err != nil {
			return err
		}
	}
	var s []byte
	b := bytes.NewBuffer(s)
	data := struct{ IsProd bool }{
		IsProd: os.Getenv("ABOT_ENV") == "production",
	}
	if err := tmplLayout.Execute(b, data); err != nil {
		return err
	}
	if err := c.HTML(http.StatusOK, string(b.Bytes())); err != nil {
		return err
	}
	return nil
}

// HandlerMain is the endpoint to hit when you want a direct response via JSON.
// The Abot console (abotc) uses this endpoint.
func HandlerMain(c *echo.Context) error {
	c.Set("cmd", c.Form("cmd"))
	c.Set("flexid", c.Form("flexid"))
	c.Set("flexidtype", c.Form("flexidtype"))
	c.Set("uid", c.Form("uid"))
	errMsg := "Something went wrong with my wiring... I'll get that fixed up soon."
	errSent := false
	ret, uid, err := ProcessText(c)
	if err != nil {
		ret = errMsg
		errSent = true
		log.Debug(err)
	}
	if err = ws.NotifySockets(c, uid, c.Form("cmd"), ret); err != nil {
		if !errSent {
			log.Debug(err)
		}
	}
	if err = c.HTML(http.StatusOK, ret); err != nil {
		if !errSent {
			log.Debug(err)
		}
	}
	return nil
}

// HandlerAPILogoutSubmit processes a logout request deleting the session from
// the server.
func HandlerAPILogoutSubmit(c *echo.Context) error {
	uid, err := util.CookieVal(c, "id")
	if err != nil {
		return JSONError(err)
	}
	if uid == "null" {
		return nil
	}
	q := `DELETE FROM sessions WHERE userid=$1`
	if _, err = db.Exec(q, uid); err != nil {
		return JSONError(err)
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerAPILoginSubmit processes a logout request deleting the session from
// the server.
func HandlerAPILoginSubmit(c *echo.Context) error {
	var req struct {
		Email    string
		Password string
	}
	if err := c.Bind(&req); err != nil {
		return JSONError(err)
	}
	var u struct {
		ID       uint64
		Password []byte
		Trainer  bool
		Admin    bool
	}
	q := `SELECT id, password, trainer, admin FROM users WHERE email=$1`
	err := db.Get(&u, q, req.Email)
	if err == sql.ErrNoRows {
		return JSONError(ErrInvalidUserPass)
	} else if err != nil {
		return JSONError(err)
	}
	if u.ID == 0 {
		return JSONError(ErrInvalidUserPass)
	}
	err = bcrypt.CompareHashAndPassword(u.Password, []byte(req.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword || err == bcrypt.ErrHashTooShort {
		return JSONError(ErrInvalidUserPass)
	} else if err != nil {
		return JSONError(err)
	}
	user := &dt.User{
		ID:      u.ID,
		Email:   req.Email,
		Trainer: u.Trainer,
		Admin:   u.Admin,
	}
	csrfToken, err := createCSRFToken(user)
	if err != nil {
		return JSONError(err)
	}
	header, token, err := getAuthToken(user)
	if err != nil {
		return JSONError(err)
	}
	resp := struct {
		ID        uint64
		Email     string
		Scopes    []string
		AuthToken string
		IssuedAt  int64
		CSRFToken string
	}{
		ID:        user.ID,
		Email:     user.Email,
		Scopes:    header.Scopes,
		AuthToken: token,
		IssuedAt:  header.IssuedAt,
		CSRFToken: csrfToken,
	}
	if err = c.JSON(http.StatusOK, resp); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerAPISignupSubmit signs up a user after server-side validation of all
// passed in values.
func HandlerAPISignupSubmit(c *echo.Context) error {
	req := struct {
		Name     string
		Email    string
		Password string
		FID      string
	}{}
	if err := c.Bind(&req); err != nil {
		return JSONError(err)
	}

	// validate the request parameters
	if len(req.Name) == 0 {
		return JSONError(errors.New("You must enter a name."))
	}
	if len(req.Email) == 0 || !strings.ContainsAny(req.Email, "@") ||
		!strings.ContainsAny(req.Email, ".") {
		return JSONError(errors.New("You must enter a valid email."))
	}
	if len(req.Password) < 8 {
		return JSONError(errors.New(
			"Your password must be at least 8 characters."))
	}
	// TODO use new SMS interface
	/*
		if err := validatePhone(req.FID); err != nil {
			return JSONError(err)
		}
	*/

	// TODO format phone number for SMS interface (international format)
	user := &dt.User{
		Name:  req.Name,
		Email: req.Email,
		// Password is hashed in user.Create()
		Password: req.Password,
		Trainer:  false,
		Admin:    false,
	}
	if err := user.Create(db, dt.FlexIDType(2), req.FID); err != nil {
		return JSONError(err)
	}
	csrfToken, err := createCSRFToken(user)
	if err != nil {
		return JSONError(err)
	}
	header, token, err := getAuthToken(user)
	if err != nil {
		return JSONError(err)
	}
	resp := struct {
		ID        uint64
		Email     string
		Scopes    []string
		AuthToken string
		IssuedAt  int64
		CSRFToken string
	}{
		ID:        user.ID,
		Email:     user.Email,
		Scopes:    []string{},
		AuthToken: token,
		IssuedAt:  header.IssuedAt,
		CSRFToken: csrfToken,
	}
	resp.ID = user.ID
	if err = c.JSON(http.StatusOK, resp); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerAPIProfile shows a user profile with the user's current addresses,
// credit cards, and contact information.
func HandlerAPIProfile(c *echo.Context) error {
	uid, err := util.CookieVal(c, "id")
	if err != nil {
		return JSONError(err)
	}
	var user struct {
		Name   string
		Email  string
		Phones []dt.Phone
		Cards  []struct {
			ID             int
			CardholderName string
			Last4          string
			ExpMonth       string `db:"expmonth"`
			ExpYear        string `db:"expyear"`
			Brand          string
		}
		Addresses []struct {
			ID      int
			Name    string
			Line1   string
			Line2   string
			City    string
			State   string
			Country string
			Zip     string
		}
	}
	q := `SELECT name, email FROM users WHERE id=$1`
	err = db.Get(&user, q, uid)
	if err != nil {
		return JSONError(err)
	}
	q = `SELECT flexid FROM userflexids
	     WHERE flexidtype=2 AND userid=$1
	     LIMIT 10`
	err = db.Select(&user.Phones, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return JSONError(err)
	}
	q = `SELECT id, cardholdername, last4, expmonth, expyear, brand
	     FROM cards
	     WHERE userid=$1
	     LIMIT 10`
	err = db.Select(&user.Cards, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return JSONError(err)
	}
	q = `SELECT id, name, line1, line2, city, state, country, zip
	     FROM addresses
	     WHERE userid=$1
	     LIMIT 10`
	err = db.Select(&user.Addresses, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return JSONError(err)
	}
	if err = c.JSON(http.StatusOK, user); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerAPIProfileView is used to validate a purchase or disclosure of
// sensitive information by a plugin. This method of validation has the user
// view their profile page, meaning that they have to be logged in on their
// device, ensuring that they either have a valid email/password or a valid
// session token in their cookies before the plugin will continue. This is a
// useful security measure because SMS is not a secure means of communication;
// SMS messages can easily be hijacked or spoofed. Taking the user to an HTTPS
// site offers the developer a better guarantee that information entered is
// coming from the correct person.
func HandlerAPIProfileView(c *echo.Context) error {
	uid, err := util.CookieVal(c, "id")
	if err != nil {
		return JSONError(err)
	}
	q := `SELECT authorizationid FROM users WHERE id=$1`
	var authID sql.NullInt64
	if err = db.Get(&authID, q, uid); err != nil {
		return JSONError(err)
	}
	if !authID.Valid {
		goto Response
	}
	q = `UPDATE authorizations SET authorizedat=$1 WHERE id=$2`
	_, err = db.Exec(q, time.Now(), authID)
	if err != nil && err != sql.ErrNoRows {
		return JSONError(err)
	}
Response:
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerAPIForgotPasswordSubmit asks the server to send the user a "Forgot
// Password" email with instructions for resetting their password.
func HandlerAPIForgotPasswordSubmit(c *echo.Context) error {
	var req struct {
		Email string
	}
	if err := c.Bind(&req); err != nil {
		return JSONError(err)
	}
	var user dt.User
	q := `SELECT id, name, email FROM users WHERE email=$1`
	err := db.Get(&user, q, req.Email)
	if err == sql.ErrNoRows {
		return JSONError(errors.New("Sorry, there's no record of that email. Are you sure that's the email you used to sign up with and that you typed it correctly?"))
	}
	if err != nil {
		return JSONError(err)
	}
	secret := RandSeq(40)
	q = `INSERT INTO passwordresets (userid, secret) VALUES ($1, $2)`
	if _, err = db.Exec(q, user.ID, secret); err != nil {
		return JSONError(err)
	}
	if len(emailsender.Drivers()) == 0 {
		return JSONError(errors.New("Sorry, this feature is not enabled. To be enabled, an email driver must be imported."))
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerAPIResetPasswordSubmit is arrived at through the email generated by
// HandlerAPIForgotPasswordSubmit. This endpoint resets the user password with
// another bcrypt hash after validating on the server that their new password is
// sufficient.
func HandlerAPIResetPasswordSubmit(c *echo.Context) error {
	var req struct {
		Secret   string
		Password string
	}
	if err := c.Bind(&req); err != nil {
		return JSONError(err)
	}
	if len(req.Password) < 8 {
		return JSONError(errors.New("Your password must be at least 8 characters"))
	}
	userid := uint64(0)
	q := `SELECT userid FROM passwordresets
	      WHERE secret=$1 AND
	            createdat >= CURRENT_TIMESTAMP - interval '30 minutes'`
	err := db.Get(&userid, q, req.Secret)
	if err == sql.ErrNoRows {
		return JSONError(errors.New("Sorry, that information doesn't match our records."))
	}
	if err != nil {
		return JSONError(err)
	}
	hpw, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return JSONError(err)
	}
	tx, err := db.Begin()
	if err != nil {
		return JSONError(err)
	}
	q = `UPDATE users SET password=$1 WHERE id=$2`
	if _, err = tx.Exec(q, hpw, userid); err != nil {
		return JSONError(err)
	}
	q = `DELETE FROM passwordresets WHERE secret=$1`
	if _, err = tx.Exec(q, req.Secret); err != nil {
		return JSONError(err)
	}
	if err = tx.Commit(); err != nil {
		return JSONError(err)
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return JSONError(err)
	}
	return nil
}

// HandlerWSConversations establishes a socket connection for the training
// interface to reload as new user messages arrive.
func HandlerWSConversations(c *echo.Context) error {
	uid, err := strconv.ParseUint(c.Query("UserID"), 10, 64)
	if err != nil {
		return err
	}
	ws.Set(uid, c.Socket())
	err = w.Message.Send(ws.Get(uid), "connected to socket")
	if err != nil {
		return err
	}
	var msg string
	for {
		// Keep the socket open
		if err = w.Message.Receive(ws.Get(uid), &msg); err != nil {
			return err
		}
	}
}

// HandlerAPIPlugins responds with all of the server's installed plugin
// configurations from each their respective plugin.json files.
func HandlerAPIPlugins(c *echo.Context) error {
	var pJSON struct {
		Plugins []json.RawMessage
	}

	// Read plugins.json, unmarshal into struct
	contents, err := ioutil.ReadFile("./plugins.json")
	if err != nil {
		return JSONError(err)
	}
	var plugins PluginJSON
	if err = json.Unmarshal(contents, &plugins); err != nil {
		return JSONError(err)
	}
	for url := range plugins.Dependencies {
		// Add each plugin.json to array of plugins
		p := filepath.Join(os.Getenv("GOPATH"), "src", url,
			"plugin.json")
		var byt []byte
		byt, err = ioutil.ReadFile(p)
		if err != nil {
			return JSONError(err)
		}
		pJSON.Plugins = append(pJSON.Plugins, byt)
	}
	if err = c.JSON(http.StatusOK, pJSON); err != nil {
		return JSONError(err)
	}
	return nil
}

// createCSRFToken creates a new token, invalidating any existing token.
func createCSRFToken(u *dt.User) (token string, err error) {
	q := `INSERT INTO sessions (token, userid, label)
	      VALUES ($1, $2, 'csrfToken')
	      ON CONFLICT (userid, label) DO UPDATE SET token=$1`
	token = RandSeq(32)
	if _, err := db.Exec(q, token, u.ID); err != nil {
		return "", err
	}
	return token, nil
}

// getAuthToken returns a token used for future client authorization with a CSRF
// token.
func getAuthToken(u *dt.User) (header *Header, authToken string, err error) {
	scopes := []string{}
	if u.Admin {
		scopes = append(scopes, "admin")
	}
	if u.Trainer {
		scopes = append(scopes, "trainer")
	}
	header = &Header{
		ID:       u.ID,
		Email:    u.Email,
		Scopes:   scopes,
		IssuedAt: time.Now().Unix(),
	}
	byt, err := json.Marshal(header)
	if err != nil {
		return nil, "", JSONError(err)
	}
	hash := hmac.New(sha512.New, []byte(os.Getenv("ABOT_SECRET")))
	_, err = hash.Write(byt)
	if err != nil {
		return nil, "", err
	}
	authToken = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return header, authToken, nil
}

// initCMDGroup establishes routes for automatically reloading the page on any
// assets/ change when a watcher is running (see cmd/*watcher.sh).
func initCMDGroup(g *echo.Group) {
	cmdch := make(chan string, 10)
	addconnch := make(chan *cmdConn, 10)
	delconnch := make(chan *cmdConn, 10)

	go cmder(cmdch, addconnch, delconnch)

	g.Get("/:cmd", func(c *echo.Context) error {
		cmdch <- c.Param("cmd")
		return c.String(http.StatusOK, "")
	})
	g.WebSocket("/ws", func(c *echo.Context) error {
		ws := c.Socket()
		respch := make(chan bool)
		conn := &cmdConn{ws: ws, respch: respch}
		addconnch <- conn
		<-respch
		delconnch <- conn
		return nil
	})
}

// cmdConn establishes a websocket and channel to listen for changes in assets/
// to automatically reload the page.
//
// To get started with autoreload, please see cmd/fswatcher.sh (cross-platform)
// or cmd/inotifywaitwatcher.sh (Linux).
type cmdConn struct {
	ws     *w.Conn
	respch chan bool
}

// cmder manages opening and closing websockets to enable autoreload on any
// assets/ change.
func cmder(cmdch <-chan string, addconnch, delconnch <-chan *cmdConn) {
	cmdconns := map[*w.Conn](chan bool){}
	for {
		select {
		case c := <-addconnch:
			cmdconns[c.ws] = c.respch
		case c := <-delconnch:
			delete(cmdconns, c.ws)
		case c := <-cmdch:
			cmd := fmt.Sprintf(`{"cmd": "%s"}`, c)
			fmt.Println("sending cmd:", cmd)
			for ws, respch := range cmdconns {
				// Error ignored because we close no matter what
				_ = w.Message.Send(ws, cmd)
				respch <- true
			}
		}
	}
}

const bearerAuthKey = "Bearer"

// Header represents an HTTP request's header from the front-end JS client. This
// is used to identify the logged in user in each web request and the
// permissions of that user.
type Header struct {
	ID       uint64
	Email    string
	Scopes   []string
	IssuedAt int64
}

// LoggedIn determines if the user is currently logged in.
func LoggedIn() echo.HandlerFunc {
	return func(c *echo.Context) error {
		log.Debug("validating logged in")
		c.Response().Header().Set(echo.WWWAuthenticate, bearerAuthKey+" realm=Restricted")
		auth := c.Request().Header.Get(echo.Authorization)
		l := len(bearerAuthKey)
		// Ensure client sent the token
		if len(auth) <= l+1 || auth[:l] != bearerAuthKey {
			log.Debug("client did not send token")
			return JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		// Ensure the token is still valid
		tmp, err := util.CookieVal(c, "issuedAt")
		if err != nil {
			return JSONError(err)
		}
		issuedAt, err := strconv.ParseInt(tmp, 10, 64)
		if err != nil {
			return JSONError(err)
		}
		t := time.Unix(issuedAt, 0)
		if t.Add(72 * time.Hour).Before(time.Now()) {
			log.Debug("token expired")
			return JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		// Ensure the token has not been tampered with
		b, err := base64.StdEncoding.DecodeString(auth[l+1:])
		if err != nil {
			return JSONError(err)
		}
		tmp, err = util.CookieVal(c, "scopes")
		if err != nil {
			return JSONError(err)
		}
		scopes := strings.Fields(tmp)
		tmp, err = util.CookieVal(c, "id")
		if err != nil {
			return JSONError(err)
		}
		userID, err := strconv.ParseUint(tmp, 10, 64)
		if err != nil {
			return JSONError(err)
		}
		email, err := util.CookieVal(c, "email")
		if err != nil {
			return JSONError(err)
		}
		a := Header{
			ID:       userID,
			Email:    email,
			Scopes:   scopes,
			IssuedAt: issuedAt,
		}
		byt, err := json.Marshal(a)
		if err != nil {
			return JSONError(err)
		}
		known := hmac.New(sha512.New, []byte(os.Getenv("ABOT_SECRET")))
		_, err = known.Write(byt)
		if err != nil {
			return JSONError(err)
		}
		ok := hmac.Equal(known.Sum(nil), b)
		if !ok {
			log.Debug("token tampered")
			return JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		log.Debug("validated logged in")
		return nil
	}
}

// CSRF ensures that any forms posted to Abot are protected against Cross-Site
// Request Forgery. Without this function, Abot would be vulnerable to the
// attack because tokens are stored client-side in cookies.
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
			return JSONError(err)
		}
		token, err := util.CookieVal(c, "csrfToken")
		if err != nil {
			return JSONError(err)
		}
		err = db.Get(&label, q, uid, token)
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
		if err != nil {
			return JSONError(err)
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
			return JSONError(err)
		}
		scopes := strings.Fields(tmp)
		for _, scope := range scopes {
			if scope == "admin" {
				log.Debug("validated admin")
				return nil
			}
		}
		return JSONError(echo.NewHTTPError(http.StatusUnauthorized))
	}
}
