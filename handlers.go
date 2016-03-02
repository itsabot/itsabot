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
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/log"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/websocket"
)

const bearerAuthKey = "Bearer"

type authHeader struct {
	ID       uint64
	Email    string
	Scopes   []string
	IssuedAt int64
}

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger(), mw.Gzip(), mw.Recover())
	//e.SetHTTPErrorHandler(handlerError)
	e.SetDebug(true)

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
	api := e.Group("/api/user", authLoggedIn(), validateCSRF())
	api.Get("/profile.json", handlerAPIProfile)
	api.Put("/profile.json", handlerAPIProfileView)

	// API routes (restricted to trainers)
	apiTrainer := e.Group("/api/trainer", authLoggedIn(), authTrainer(),
		validateCSRF())
	apiTrainer.Get("/message.json", handlerAPIConversationsShow)
	apiTrainer.Get("/messages.json", handlerAPIMessages)
	apiTrainer.Get("/contacts/search.json", handlerAPIContactsSearch)
	apiTrainer.Post("/trigger.json", handlerAPITriggerPkg)
	apiTrainer.Post("/messages.json", handlerAPIMessagesCreate)
	apiTrainer.Patch("/conversation.json", handlerAPIConversationsComplete)
	apiTrainer.Post("/contacts/conversations.json",
		handlerAPIContactsConversationsCreate)

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
		handlerError(err, c)
	}
	if err = ws.NotifySockets(c, uid, c.Form("cmd"), ret); err != nil {
		if !errSent {
			handlerError(err, c)
		}
	}
	if err = c.HTML(http.StatusOK, ret); err != nil {
		if !errSent {
			handlerError(err, c)
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
	pkg, route, _, err := core.GetPkg(db, msg)
	if err != nil {
		log.Debug("could not get core package", err)
		return core.JSONError(err)
	}
	msg.Route = route
	if pkg == nil {
		msg.Package = ""
	} else {
		msg.Package = pkg.P.Config.Name
	}
	ret, err := core.CallPkg(pkg, msg, false)
	if err != nil {
		log.Debug("could not call package", err)
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
		m.Package = pkg.P.Config.Name
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
	uid, err := cookieVal(c, "id")
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
	// TODO createCSRFToken
	var resp struct {
		ID        uint64
		Email     string
		Scopes    []string
		AuthToken string
		IssuedAt  uint64
		CSRFToken string
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
	uid, err := cookieVal(c, "id")
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
// sensitive information by a package. This method of validation has the user
// view their profile page, meaning that they have to be logged in on their
// device, ensuring that they either have a valid email/password or a valid
// session token in their cookies before the package will continue. This is a
// useful security measure because SMS is not a secure means of communication;
// SMS messages can easily be hijacked or spoofed. Taking the user to an HTTPS
// site offers the developer a better guarantee that information entered is
// coming from the correct person.
func handlerAPIProfileView(c *echo.Context) error {
	uid, err := cookieVal(c, "id")
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
	// TODO implement mail interface
	/*
		h := template.ForgotPasswordEmail(user.Name, secret)
		if err := mc.Send("Password reset", h, &user); err != nil {
			return core.JSONError(err)
		}
	*/
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

// handlerAPIMessages loads up conversations that need training for the
// Training Index endpoint. A max of 1 message per user will be loaded, since
// any user that needs help will receive help for their most recent request
// via their most recent message.
func handlerAPIMessages(c *echo.Context) error {
	var msgs []struct {
		ID        uint64
		Sentence  string
		UserID    uint64
		CreatedAt *time.Time
	}
	q := `SELECT DISTINCT ON (userid)
	          id, sentence, createdat, userid
	      FROM messages
	      WHERE needstraining IS TRUE
	      ORDER BY userid, createdat DESC
	      LIMIT 25`
	err := db.Select(&msgs, q)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	if err = c.JSON(http.StatusOK, msgs); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPIConversationsComplete marks a conversation as complete, so it's no
// longer presented in the Training Index.
func handlerAPIConversationsComplete(c *echo.Context) error {
	uid, err := strconv.Atoi(c.Query("uid"))
	if err != nil {
		return core.JSONError(err)
	}
	q := `UPDATE messages SET needstraining=FALSE WHERE userid=$1`
	_, err = db.Exec(q, uid)
	return err
}

// handlerAPIMessagesShow returns all relevant messages and information in a
// single conversation to enable trainers to get a sense of what happened in the
// messages leading up to this problem and provide a better and faster solution.
func handlerAPIConversationsShow(c *echo.Context) error {
	uid, err := cookieVal(c, "id")
	if err != nil {
		return core.JSONError(err)
	}
	var ret []struct {
		Sentence  string
		AbotSent  bool
		CreatedAt time.Time
	}
	q := `SELECT sentence, abotsent, createdat
	      FROM messages
	      WHERE userid=$1
	      ORDER BY createdat DESC
	      LIMIT 10`
	if err := db.Select(&ret, q, uid); err != nil {
		return core.JSONError(err)
	}
	var username string
	q = `SELECT name FROM users WHERE id=$1`
	if err := db.Get(&username, q, uid); err != nil {
		return core.JSONError(err)
	}
	// reverse order of messages for display
	for i, j := 0, len(ret)-1; i < j; i, j = i+1, j-1 {
		ret[i], ret[j] = ret[j], ret[i]
	}
	q = `SELECT DISTINCT ON (key) key, value
	     FROM preferences
	     WHERE userid=$1
	     ORDER BY key, createdat DESC
	     LIMIT 25`
	var prefsTmp []struct {
		Key   string
		Value string
	}
	err = db.Select(&prefsTmp, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	var prefs []string
	for _, p := range prefsTmp {
		prefs = append(prefs, p.Key+": "+
			strings.ToUpper(p.Value[:1])+p.Value[1:])
	}
	var tmp []string
	q = `SELECT label FROM sessions
	     WHERE label='gcal_token' AND userid=$1 AND token IS NOT NULL`
	err = db.Select(&tmp, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	cals := []string{}
	if len(tmp) > 0 {
		cals = append(cals, "Google")
	}
	var addrsTmp []struct {
		Name  string
		Line1 string
	}
	q = `SELECT name, line1 FROM addresses WHERE userid=$1`
	err = db.Select(&addrsTmp, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	var addrs []string
	for _, addr := range addrsTmp {
		if len(addr.Name) > 0 {
			s := fmt.Sprintf("%s (%s)", addr.Name, addr.Line1)
			addrs = append(addrs, s)
		} else {
			addrs = append(addrs, addr.Line1)
		}
	}
	var cardsTmp []struct {
		Last4 string
		Brand string
	}
	q = `SELECT brand, last4 FROM cards WHERE userid=$1`
	err = db.Select(&cardsTmp, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return core.JSONError(err)
	}
	var cards []string
	for _, card := range cardsTmp {
		s := fmt.Sprintf("%s (%s)", card.Brand, card.Last4)
		cards = append(cards, s)
	}
	resp := struct {
		Username string
		Chats    []struct {
			Sentence  string
			AbotSent  bool
			CreatedAt time.Time
		}
		Preferences []string
		Calendars   []string
		Addresses   []string
		Cards       []string
	}{
		Username:    username,
		Chats:       ret,
		Preferences: prefs,
		Calendars:   cals,
		Addresses:   addrs,
		Cards:       cards,
	}
	if err := c.JSON(http.StatusOK, resp); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// handlerAPIMessagesCreate sends a message to a user on behalf of Ava and is
// called via the training interface.
func handlerAPIMessagesCreate(c *echo.Context) error {
	var req struct {
		Sentence string
		UserID   uint64
	}
	if err := c.Bind(&req); err != nil {
		return core.JSONError(err)
	}
	// TODO record the last flextype used and send the user a response via
	// that. e.g. if message was received from a secondary phone number,
	// respond to the user via that secondary phone number. For now, simply
	// get the first flexid available
	var fid string
	q := `SELECT flexid FROM userflexids WHERE flexidtype=2 AND userid=$1`
	if err := db.Get(&fid, q, req.UserID); err != nil {
		return core.JSONError(err)
	}
	var id uint64
	q = `INSERT INTO messages
	     (userid, sentence, abotsent) VALUES ($1, $2, TRUE) RETURNING id`
	err := db.QueryRowx(q, req.UserID, req.Sentence).Scan(&id)
	if err != nil {
		return core.JSONError(err)
	}
	/*
		if err = sms.SendMessage(tc, fid, req.Sentence); err != nil {
			q = `DELETE FROM messages WHERE id=$1`
			if _, err = db.Exec(q, id); err != nil {
				return core.JSONError(err)
			}
			return core.JSONError(err)
		}
	*/
	if err := c.JSON(http.StatusOK, nil); err != nil {
		return core.JSONError(err)
	}
	return nil
}

// TODO
// handlerAPIContactsConversationsCreate is not yet implemented. At this point
// we're finalizing the Contact API before continuing.
func handlerAPIContactsConversationsCreate(c *echo.Context) error {
	var req struct {
		Sentence      string
		Contact       string
		ContactMethod string
	}
	if err := c.Bind(&req); err != nil {
		return core.JSONError(err)
	}
	// TODO insert into contact's messages, send message
	return core.JSONError(errors.New("ContactsConversationsCreate not implemented"))
}

// handlerAPIContactsSearch provides a way to query for contacts using full-text
// search on their name, email and phone number.
//
// TODO this will be implemented when the Contact API has been decided.
func handlerAPIContactsSearch(c *echo.Context) error {
	/*
		uid, err := strconv.Atoi(c.Query("UserID"))
		if err != nil {
			return core.JSONError(err)
		}
		var results []dt.Contact
		q := `SELECT name, email, phone FROM contacts
		      WHERE userid=$1 AND name ILIKE $2`
		term := "%" + c.Query("Query") + "%"
		if err := db.Select(&results, q, uid, term); err != nil {
			return core.JSONError(err)
		}
		if err := c.JSON(http.StatusOK, results); err != nil {
			return core.JSONError(err)
		}
	*/
	return core.JSONError(errors.New("not implemented"))
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

func handlerError(err error, c *echo.Context) {
	log.Debug("failed handling", err)
	// TODO implement mail interface
	// mc.SendBug(err)
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
func getAuthToken(u *dt.User) (header *authHeader, authToken string,
	err error) {

	scopes := []string{}
	if u.Trainer {
		scopes = append(scopes, "trainer")
	}
	header = &authHeader{
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

func authLoggedIn() echo.HandlerFunc {
	return func(c *echo.Context) error {
		log.Debug("validating logged in")
		// Skip WebSocket
		if (c.Request().Header.Get(echo.Upgrade)) == echo.WebSocket {
			return nil
		}
		c.Response().Header().Set(echo.WWWAuthenticate, bearerAuthKey+" realm=Restricted")
		auth := c.Request().Header.Get(echo.Authorization)
		l := len(bearerAuthKey)
		// Ensure client sent the token
		if len(auth) <= l+1 || auth[:l] != bearerAuthKey {
			log.Debug("client did not send token")
			return core.JSONError(echo.NewHTTPError(http.StatusUnauthorized))
		}
		// Ensure the token is still valid
		tmp, err := cookieVal(c, "issuedAt")
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
		tmp, err = cookieVal(c, "scopes")
		if err != nil {
			return core.JSONError(err)
		}
		scopes := strings.Fields(tmp)
		tmp, err = cookieVal(c, "id")
		if err != nil {
			return core.JSONError(err)
		}
		userID, err := strconv.ParseUint(tmp, 10, 64)
		if err != nil {
			return core.JSONError(err)
		}
		email, err := cookieVal(c, "email")
		if err != nil {
			return core.JSONError(err)
		}
		a := authHeader{
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

func authTrainer() echo.HandlerFunc {
	return func(c *echo.Context) error {
		// Since the headers are validated in authLoggedIn, we can trust
		// these scopes and can avoid a DB request
		log.Debug("validating trainer")
		tmp, err := cookieVal(c, "scopes")
		if err != nil {
			return core.JSONError(err)
		}
		scopes := strings.Fields(tmp)
		for _, scope := range scopes {
			if scope == "trainer" {
				log.Debug("validated trainer")
				return nil
			}
		}
		return echo.NewHTTPError(http.StatusUnauthorized)
	}
}

// validateCSRF ensures that any forms posted to Abot are protected against
// Cross-Site Request Forgery. Without this function, Abot would be vulnerable
// to the attack because tokens are stored client-side in cookies.
func validateCSRF() echo.HandlerFunc {
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
		uid, err := cookieVal(c, "id")
		if err != nil {
			return core.JSONError(err)
		}
		token, err := cookieVal(c, "csrfToken")
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

// cookieVal retrieves a cookie
func cookieVal(c *echo.Context, name string) (value string, err error) {
	ck, err := c.Request().Cookie(name)
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, name)
	}
	val, err := url.QueryUnescape(ck.Value)
	if err != nil {
		return "", err
	}
	return val, nil
}
