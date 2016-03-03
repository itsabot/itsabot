package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/auth"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/interface/emailsender"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/util"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/websocket"
)

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger(), mw.Gzip(), mw.Recover())
	e.SetDebug(true)
	logger := log.New("")
	e.SetLogger(logger)

	e.Static("/public/css", "public/css")
	e.Static("/public/js", "public/js")
	e.Static("/public/images", "assets/images")

	if os.Getenv("ABOT_ENV") != "production" {
		cmd := e.Group("/_/cmd")
		initCMDGroup(cmd)
	}

	// Web routes
	e.Get("/*", handlerIndex)
	e.Post("/", handlerMain)

	// API routes (no restrictions)
	e.Post("/api/login.json", handlerAPILoginSubmit)
	e.Post("/api/logout.json", handlerAPILogoutSubmit)
	e.Post("/api/signup.json", handlerAPISignupSubmit)
	e.Post("/api/forgot_password.json", handlerAPIForgotPasswordSubmit)
	e.Post("/api/reset_password.json", handlerAPIResetPasswordSubmit)

	// API routes (restricted by login)
	api := e.Group("/api/user", auth.LoggedIn(), auth.CSRF(db))
	api.Get("/profile.json", handlerAPIProfile)
	api.Put("/profile.json", handlerAPIProfileView)

	// API routes (restricted to admins)
	apiAdmin := e.Group("/api/admin", auth.LoggedIn(), auth.CSRF(db),
		auth.Admin())
	apiAdmin.Get("/plugins.json", handlerAPIPlugins)

	// WebSockets
	e.WebSocket("/ws", handlerWSConversations)
}

// CMDConn establishes a websocket and channel to listen for changes in assets/
// to automatically reload the page.
//
// To get started with autoreload, please see cmd/fswatcher.sh (cross-platform)
// or cmd/inotifywaitwatcher.sh (Linux).
type CMDConn struct {
	ws     *websocket.Conn
	respch chan bool
}

// cmder manages opening and closing websockets to enable autoreload on any
// assets/ change.
func cmder(cmdch <-chan string, addconnch, delconnch <-chan *CMDConn) {
	cmdconns := map[*websocket.Conn](chan bool){}
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
				_ = websocket.Message.Send(ws, cmd)
				respch <- true
			}
		}
	}
}

// initCMDGroup establishes routes for automatically reloading the page on any
// assets/ change when a watcher is running (see cmd/*watcher.sh).
func initCMDGroup(g *echo.Group) {
	cmdch := make(chan string, 10)
	addconnch := make(chan *CMDConn, 10)
	delconnch := make(chan *CMDConn, 10)

	go cmder(cmdch, addconnch, delconnch)

	g.Get("/:cmd", func(c *echo.Context) error {
		cmdch <- c.Param("cmd")
		return c.String(http.StatusOK, "")
	})
	g.WebSocket("/ws", func(c *echo.Context) error {
		ws := c.Socket()
		respch := make(chan bool)
		conn := &CMDConn{ws: ws, respch: respch}
		addconnch <- conn
		<-respch
		delconnch <- conn
		return nil
	})
}

// handlerIndex presents the homepage to the user and populates the HTML with
// server-side variables.
func handlerIndex(c *echo.Context) error {
	// TODO split out to main unless in development
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		log.Fatal(err)
	}
	var s []byte
	b := bytes.NewBuffer(s)
	data := struct{ IsProd bool }{
		IsProd: os.Getenv("ABOT_ENV") == "production",
	}
	if err := tmplLayout.Execute(b, data); err != nil {
		return err
	}
	if err = c.HTML(http.StatusOK, string(b.Bytes())); err != nil {
		return err
	}
	return nil
}

// handlerMain is the endpoint to hit when you want a direct response via JSON.
// The Abot console (abotc) uses this endpoint.
func handlerMain(c *echo.Context) error {
	c.Set("cmd", c.Form("cmd"))
	c.Set("flexid", c.Form("flexid"))
	c.Set("flexidtype", c.Form("flexidtype"))
	c.Set("uid", c.Form("uid"))
	errMsg := "Something went wrong with my wiring... I'll get that fixed up soon."
	errSent := false
	ret, uid, err := core.ProcessText(db, ner, offensive, c)
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

// handlerAPITriggerPkg enables easier communication via JSON with the training
// interface when trainers want to "trigger" an action on behalf of a user.
func handlerAPITriggerPkg(c *echo.Context) error {
	c.Set("cmd", c.Form("cmd"))
	c.Set("uid", c.Form("uid"))
	msg, err := core.Preprocess(db, ner, c)
	if err != nil {
		return core.JSONError(err)
	}
	pkg, route, _, err := core.GetPlugin(db, msg)
	if err != nil {
		log.Debug("could not get core plugin", err)
		return core.JSONError(err)
	}
	msg.Route = route
	if pkg == nil {
		msg.Plugin = ""
	} else {
		msg.Plugin = pkg.P.Config.Name
	}
	ret, err := core.CallPlugin(pkg, msg, false)
	if err != nil {
		log.Debug("could not call plugin", err)
		return core.JSONError(err)
	}
	if len(ret) == 0 {
		tmp := fmt.Sprintf("%s %s", "missing trigger/pkg for cmd",
			c.Get("cmd"))
		return core.JSONError(errors.New(tmp))
	}
	m := &dt.Msg{}
	m.AbotSent = true
	m.User = msg.User
	m.Sentence = ret
	if pkg != nil {
		m.Plugin = pkg.P.Config.Name
	}
	if err = m.Save(db); err != nil {
		log.Debug("could not save Abot response message", err)
		return core.JSONError(err)
	}
	resp := struct {
		Msg string
	}{Msg: ret}
	if err = c.JSON(http.StatusOK, resp); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPILogoutSubmit processes a logout request deleting the session from
// the server.
func handlerAPILogoutSubmit(c *echo.Context) error {
	uid, err := util.CookieVal(c, "id")
	if err != nil {
		return core.JSONError(err)
	}
	if uid == "null" {
		return nil
	}
	q := `DELETE FROM sessions WHERE userid=$1`
	if _, err = db.Exec(q, uid); err != nil {
		return core.JSONError(err)
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPILoginSubmit processes a logout request deleting the session from
// the server.
func handlerAPILoginSubmit(c *echo.Context) error {
	var req struct {
		Email    string
		Password string
	}
	if err := c.Bind(&req); err != nil {
		return core.JSONError(err)
	}
	var u struct {
		ID       uint64
		Password []byte
		Trainer  bool
	}
	q := `SELECT id, password, trainer FROM users WHERE email=$1`
	err := db.Get(&u, q, req.Email)
	if err == sql.ErrNoRows {
		return core.JSONError(errInvalidUserPass)
	} else if err != nil {
		return core.JSONError(err)
	}
	if u.ID == 0 {
		return core.JSONError(errInvalidUserPass)
	}
	err = bcrypt.CompareHashAndPassword(u.Password, []byte(req.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword || err == bcrypt.ErrHashTooShort {
		return core.JSONError(errInvalidUserPass)
	} else if err != nil {
		return core.JSONError(err)
	}
	user := &dt.User{
		ID:      u.ID,
		Email:   req.Email,
		Trainer: u.Trainer,
	}
	csrfToken, err := createCSRFToken(user)
	if err != nil {
		return core.JSONError(err)
	}
	header, token, err := getAuthToken(user)
	if err != nil {
		return core.JSONError(err)
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
		return core.JSONError(err)
	}
	return nil
}

// handlerAPISignupSubmit signs up a user after server-side validation of all
// passed in values.
func handlerAPISignupSubmit(c *echo.Context) error {
	req := struct {
		Name     string
		Email    string
		Password string
		FID      string
	}{}
	if err := c.Bind(&req); err != nil {
		return core.JSONError(err)
	}

	// validate the request parameters
	if len(req.Name) == 0 {
		return core.JSONError(errors.New("You must enter a name."))
	}
	if len(req.Email) == 0 || !strings.ContainsAny(req.Email, "@") ||
		!strings.ContainsAny(req.Email, ".") {
		return core.JSONError(errors.New("You must enter a valid email."))
	}
	if len(req.Password) < 8 {
		return core.JSONError(errors.New(
			"Your password must be at least 8 characters."))
	}
	// TODO use new SMS interface
	/*
		if err := validatePhone(req.FID); err != nil {
			return core.JSONError(err)
		}
	*/

	// create the password hash
	// TODO format phone number for SMS interface (international format)
	hpw, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return core.JSONError(err)
	}

	tx, err := db.Beginx()
	if err != nil {
		return core.JSONError(errors.New("Something went wrong. Try again."))
	}
	q := `INSERT INTO users (name, email, password, locationid)
	      VALUES ($1, $2, $3, 0)
	      RETURNING id`
	var uid uint64
	err = tx.QueryRowx(q, req.Name, req.Email, hpw).Scan(&uid)
	if err != nil && err.Error() ==
		`pq: duplicate key value violates unique constraint "users_email_key"` {
		_ = tx.Rollback()
		return core.JSONError(errors.New("Sorry, that email is taken."))
	}
	if uid == 0 {
		_ = tx.Rollback()
		return core.JSONError(errors.New(
			"Something went wrong. Please try again."))
	}
	q = `INSERT INTO userflexids (userid, flexid, flexidtype)
	     VALUES ($1, $2, $3)`
	_, err = tx.Exec(q, uid, req.FID, 2)
	if err != nil {
		_ = tx.Rollback()
		return core.JSONError(errors.New(
			"Couldn't sign up. Did you use the link sent to you?"))
	}
	if err = tx.Commit(); err != nil {
		return core.JSONError(errors.New(
			"Something went wrong. Please try again."))
	}
	user := &dt.User{
		ID:      uid,
		Email:   req.Email,
		Trainer: false,
	}
	csrfToken, err := createCSRFToken(user)
	if err != nil {
		return core.JSONError(err)
	}
	header, token, err := getAuthToken(user)
	if err != nil {
		return core.JSONError(err)
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
	resp.ID = uid
	if os.Getenv("ABOT_ENV") == "production" {
		fName := strings.Fields(req.Name)[0]
		msg := fmt.Sprintf("Nice to meet you, %s. ", fName)
		msg += "How can I help? Try asking me to help you find a nice bottle of wine."
		// TODO move to the new SMS interface
		/*
			if err = sms.SendMessage(tc, req.FID, msg); err != nil {
				return core.JSONError(err)
			}
		*/
	}
	if err = c.JSON(http.StatusOK, resp); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPIProfile shows a user profile with the user's current addresses,
// credit cards, and contact information.
func handlerAPIProfile(c *echo.Context) error {
	uid, err := util.CookieVal(c, "id")
	if err != nil {
		return core.JSONError(err)
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
		return core.JSONError(err)
	}
	q = `SELECT flexid FROM userflexids
	     WHERE flexidtype=2 AND userid=$1
	     LIMIT 10`
	err = db.Select(&user.Phones, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	q = `SELECT id, cardholdername, last4, expmonth, expyear, brand
	     FROM cards
	     WHERE userid=$1
	     LIMIT 10`
	err = db.Select(&user.Cards, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	q = `SELECT id, name, line1, line2, city, state, country, zip
	     FROM addresses
	     WHERE userid=$1
	     LIMIT 10`
	err = db.Select(&user.Addresses, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	if err = c.JSON(http.StatusOK, user); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPIProfileView is used to validate a purchase or disclosure of
// sensitive information by a plugin. This method of validation has the user
// view their profile page, meaning that they have to be logged in on their
// device, ensuring that they either have a valid email/password or a valid
// session token in their cookies before the plugin will continue. This is a
// useful security measure because SMS is not a secure means of communication;
// SMS messages can easily be hijacked or spoofed. Taking the user to an HTTPS
// site offers the developer a better guarantee that information entered is
// coming from the correct person.
func handlerAPIProfileView(c *echo.Context) error {
	uid, err := util.CookieVal(c, "id")
	if err != nil {
		return core.JSONError(err)
	}
	q := `SELECT authorizationid FROM users WHERE id=$1`
	var authID sql.NullInt64
	if err = db.Get(&authID, q, uid); err != nil {
		return core.JSONError(err)
	}
	if !authID.Valid {
		goto Response
	}
	q = `UPDATE authorizations SET authorizedat=$1 WHERE id=$2`
	_, err = db.Exec(q, time.Now(), authID)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
Response:
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPIForgotPasswordSubmit asks the server to send the user a "Forgot
// Password" email with instructions for resetting their password.
func handlerAPIForgotPasswordSubmit(c *echo.Context) error {
	var req struct {
		Email string
	}
	if err := c.Bind(&req); err != nil {
		return core.JSONError(err)
	}
	var user dt.User
	q := `SELECT id, name, email FROM users WHERE email=$1`
	err := db.Get(&user, q, req.Email)
	if err == sql.ErrNoRows {
		return core.JSONError(errors.New("Sorry, there's no record of that email. Are you sure that's the email you used to sign up with and that you typed it correctly?"))
	}
	if err != nil {
		return core.JSONError(err)
	}
	secret := randSeq(40)
	q = `INSERT INTO passwordresets (userid, secret) VALUES ($1, $2)`
	if _, err := db.Exec(q, user.ID, secret); err != nil {
		return core.JSONError(err)
	}
	if len(emailsender.Drivers()) == 0 {
		return core.JSONError(errors.New("Sorry, this feature is not enabled. To be enabled, an email driver must be imported."))
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPIResetPasswordSubmit is arrived at through the email generated by
// handlerAPIForgotPasswordSubmit. This endpoint resets the user password with
// another bcrypt hash after validating on the server that their new password is
// sufficient.
func handlerAPIResetPasswordSubmit(c *echo.Context) error {
	var req struct {
		Secret   string
		Password string
	}
	if err := c.Bind(&req); err != nil {
		return core.JSONError(err)
	}
	if len(req.Password) < 8 {
		return core.JSONError(errors.New("Your password must be at least 8 characters"))
	}
	userid := uint64(0)
	q := `SELECT userid FROM passwordresets
	      WHERE secret=$1 AND
	            createdat >= CURRENT_TIMESTAMP - interval '30 minutes'`
	err := db.Get(&userid, q, req.Secret)
	if err == sql.ErrNoRows {
		return core.JSONError(errors.New("Sorry, that information doesn't match our records."))
	}
	if err != nil {
		return core.JSONError(err)
	}
	hpw, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return core.JSONError(err)
	}
	tx, err := db.Begin()
	if err != nil {
		return core.JSONError(err)
	}
	q = `UPDATE users SET password=$1 WHERE id=$2`
	if _, err = tx.Exec(q, hpw, userid); err != nil {
		return core.JSONError(err)
	}
	q = `DELETE FROM passwordresets WHERE secret=$1`
	if _, err = tx.Exec(q, req.Secret); err != nil {
		return core.JSONError(err)
	}
	if err = tx.Commit(); err != nil {
		return core.JSONError(err)
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerWSConversations establishes a socket connection for the training
// interface to reload as new user messages arrive.
func handlerWSConversations(c *echo.Context) error {
	uid, err := strconv.ParseUint(c.Query("UserID"), 10, 64)
	if err != nil {
		return err
	}
	ws.Set(uid, c.Socket())
	err = websocket.Message.Send(ws.Get(uid), "connected to socket")
	if err != nil {
		return err
	}
	var msg string
	for {
		// Keep the socket open
		if err = websocket.Message.Receive(ws.Get(uid), &msg); err != nil {
			return err
		}
	}
	return nil
}

func handlerAPIPlugins(c *echo.Context) error {
	// Get all files in the plugins directory
	fis, err := ioutil.ReadDir("./plugins")
	if err != nil {
		return core.JSONError(err)
	}
	var pJSON struct {
		Plugins []json.RawMessage
	}
	for _, fi := range fis {
		// Skip anything that's not a directory
		if !fi.IsDir() {
			continue
		}
		// Add each plugin.json to array of plugins
		p := filepath.Join("./plugins", fi.Name(), "plugin.json")
		byt, err := ioutil.ReadFile(p)
		if err != nil {
			return core.JSONError(err)
		}
		pJSON.Plugins = append(pJSON.Plugins, byt)
	}
	if err = c.JSON(http.StatusOK, pJSON); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// createCSRFToken creates a new token, invalidating any existing token.
func createCSRFToken(u *dt.User) (token string, err error) {
	q := `INSERT INTO sessions (token, userid, label)
	      VALUES ($1, $2, 'csrfToken')
	      ON CONFLICT (userid, label) DO UPDATE SET token=$1`
	token = randSeq(32)
	if _, err := db.Exec(q, token, u.ID); err != nil {
		return "", err
	}
	return token, nil
}

// getAuthToken returns a token used for future client authorization with a CSRF
// token.
func getAuthToken(u *dt.User) (header *auth.Header, authToken string,
	err error) {

	scopes := []string{}
	if u.Trainer {
		scopes = append(scopes, "trainer")
	}
	header = &auth.Header{
		ID:       u.ID,
		Email:    u.Email,
		Scopes:   scopes,
		IssuedAt: time.Now().Unix(),
	}
	byt, err := json.Marshal(header)
	if err != nil {
		return nil, "", core.JSONError(err)
	}
	hash := hmac.New(sha512.New, []byte(os.Getenv("ABOT_SECRET")))
	_, err = hash.Write(byt)
	if err != nil {
		return nil, "", err
	}
	authToken = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return header, authToken, nil
}
