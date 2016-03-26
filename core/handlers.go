package core

import (
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

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/core/websocket"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/interface/emailsender"
	"github.com/julienschmidt/httprouter"
)

var tmplLayout *template.Template
var ws = websocket.NewAtomicWebSocketSet()

// ErrInvalidUserPass reports an invalid username/password combination during
// login.
var ErrInvalidUserPass = errors.New("Invalid username/password combination")

// newRouter initializes and returns a router.
func newRouter() *httprouter.Router {
	router := httprouter.New()
	router.ServeFiles("/public/*filepath", http.Dir("public"))

	if os.Getenv("ABOT_ENV") != "production" {
		initCMDGroup(router)
	}

	// Web routes
	router.HandlerFunc("GET", "/", HIndex)
	router.HandlerFunc("POST", "/", HMain)

	// Route any unknown request to our single page app front-end
	router.NotFound = http.HandlerFunc(HIndex)

	// API routes (no restrictions)
	router.HandlerFunc("POST", "/api/login.json", HAPILoginSubmit)
	router.HandlerFunc("POST", "/api/logout.json", HAPILogoutSubmit)
	router.HandlerFunc("POST", "/api/signup.json", HAPISignupSubmit)
	router.HandlerFunc("POST", "/api/forgot_password.json", HAPIForgotPasswordSubmit)
	router.HandlerFunc("POST", "/api/reset_password.json", HAPIResetPasswordSubmit)

	// API routes (restricted by login)
	router.HandlerFunc("GET", "/api/user/profile.json", HAPIProfile)
	router.HandlerFunc("PUT", "/api/user/profile.json", HAPIProfileView)

	// API routes (restricted to admins)
	router.HandlerFunc("GET", "/api/admin/plugins.json", HAPIPlugins)
	return router
}

// HIndex presents the homepage to the user and populates the HTML with
// server-side variables.
func HIndex(w http.ResponseWriter, r *http.Request) {
	var err error
	if os.Getenv("ABOT_ENV") != "development" {
		p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com",
			"itsabot", "abot", "assets", "html", "layout.html")
		tmplLayout, err = template.ParseFiles(p)
		if err != nil {
			writeErrorInternal(w, err)
			return
		}
		if err = CompileAssets(); err != nil {
			writeErrorInternal(w, err)
			return
		}
	}
	data := struct{ IsProd bool }{
		IsProd: os.Getenv("ABOT_ENV") == "production",
	}
	if err = tmplLayout.Execute(w, data); err != nil {
		writeErrorInternal(w, err)
	}
}

// HMain is the endpoint to hit when you want a direct response via JSON.
// The Abot console (abotc) uses this endpoint.
func HMain(w http.ResponseWriter, r *http.Request) {
	errMsg := "Something went wrong with my wiring... I'll get that fixed up soon."
	ret, _, err := ProcessText(r)
	if err != nil {
		ret = errMsg
		log.Debug(err)
		// TODO notify plugins listening for errors
	}
	_, err = fmt.Fprint(w, ret)
	if err != nil {
		writeErrorInternal(w, err)
	}
}

// HAPILogoutSubmit processes a logout request deleting the session from
// the server.
func HAPILogoutSubmit(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("id")
	if err != nil {
		writeError(w, err)
		return
	}
	uid := cookie.Value
	if uid == "null" {
		http.Error(w, "id was null", http.StatusBadRequest)
		return
	}
	q := `DELETE FROM sessions WHERE userid=$1`
	if _, err = db.Exec(q, uid); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HAPILoginSubmit processes a logout request deleting the session from
// the server.
func HAPILoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, err)
		return
	}
	var u struct {
		ID       uint64
		Password []byte
		Trainer  bool
		Admin    bool
	}
	q := `SELECT id, password, trainer, admin FROM users WHERE email=$1`
	err := db.Get(&u, q, r.FormValue("Email"))
	if err == sql.ErrNoRows {
		writeError(w, ErrInvalidUserPass)
		return
	} else if err != nil {
		writeError(w, err)
		return
	}
	if u.ID == 0 {
		writeError(w, ErrInvalidUserPass)
		return
	}
	err = bcrypt.CompareHashAndPassword(u.Password, []byte(r.FormValue("Password")))
	if err == bcrypt.ErrMismatchedHashAndPassword || err == bcrypt.ErrHashTooShort {
		writeError(w, ErrInvalidUserPass)
		return
	} else if err != nil {
		writeError(w, err)
		return
	}
	user := &dt.User{
		ID:      u.ID,
		Email:   r.FormValue("Email"),
		Trainer: u.Trainer,
		Admin:   u.Admin,
	}
	csrfToken, err := createCSRFToken(user)
	if err != nil {
		writeError(w, err)
		return
	}
	header, token, err := getAuthToken(user)
	if err != nil {
		writeError(w, err)
		return
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
	writeBytes(w, resp)
}

// HAPISignupSubmit signs up a user after server-side validation of all
// passed in values.
func HAPISignupSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, err)
		return
	}

	// validate the request parameters
	if len(r.FormValue("Name")) == 0 {
		writeError(w, errors.New("You must enter a name."))
		return
	}
	if len(r.FormValue("Email")) == 0 ||
		!strings.ContainsAny(r.FormValue("Email"), "@") ||
		!strings.ContainsAny(r.FormValue("Email"), ".") {
		writeError(w, errors.New("You must enter a valid email."))
		return
	}
	if len(r.FormValue("Password")) < 8 {
		writeError(w, errors.New("Your password must be at least 8 characters."))
		return
	}
	// TODO use new SMS interface
	/*
		if err := validatePhone(req.FID); err != nil {
			return JSONError(err)
		}
	*/

	// TODO format phone number for SMS interface (international format)
	user := &dt.User{
		Name:  r.FormValue("Name"),
		Email: r.FormValue("Email"),
		// Password is hashed in user.Create()
		Password: r.FormValue("Password"),
		Trainer:  false,
		Admin:    false,
	}
	err := user.Create(db, dt.FlexIDType(2), r.FormValue("FID"))
	if err != nil {
		writeError(w, err)
		return
	}
	csrfToken, err := createCSRFToken(user)
	if err != nil {
		writeError(w, err)
		return
	}
	header, token, err := getAuthToken(user)
	if err != nil {
		writeError(w, err)
		return
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
	writeBytes(w, resp)
}

// HAPIProfile shows a user profile with the user's current addresses, credit
// cards, and contact information.
func HAPIProfile(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ABOT_ENV") != "test" {
		if !LoggedIn(w, r) {
			return
		}
	}

	cookie, err := r.Cookie("id")
	if err != nil {
		writeError(w, err)
		return
	}
	uid := cookie.Value
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
		writeError(w, err)
		return
	}
	q = `SELECT flexid FROM userflexids
	     WHERE flexidtype=2 AND userid=$1
	     LIMIT 10`
	err = db.Select(&user.Phones, q, uid)
	if err != nil && err != sql.ErrNoRows {
		writeError(w, err)
		return
	}
	q = `SELECT id, cardholdername, last4, expmonth, expyear, brand
	     FROM cards
	     WHERE userid=$1
	     LIMIT 10`
	err = db.Select(&user.Cards, q, uid)
	if err != nil && err != sql.ErrNoRows {
		writeError(w, err)
		return
	}
	q = `SELECT id, name, line1, line2, city, state, country, zip
	     FROM addresses
	     WHERE userid=$1
	     LIMIT 10`
	err = db.Select(&user.Addresses, q, uid)
	if err != nil && err != sql.ErrNoRows {
		writeError(w, err)
		return
	}
	writeBytes(w, user)
}

// HAPIProfileView is used to validate a purchase or disclosure of
// sensitive information by a plugin. This method of validation has the user
// view their profile page, meaning that they have to be logged in on their
// device, ensuring that they either have a valid email/password or a valid
// session token in their cookies before the plugin will continue. This is a
// useful security measure because SMS is not a secure means of communication;
// SMS messages can easily be hijacked or spoofed. Taking the user to an HTTPS
// site offers the developer a better guarantee that information entered is
// coming from the correct person.
func HAPIProfileView(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ABOT_ENV") != "test" {
		if !LoggedIn(w, r) {
			return
		}
		if !CSRF(w, r) {
			return
		}
	}

	cookie, err := r.Cookie("id")
	if err != nil {
		writeErrorInternal(w, err)
		return
	}
	uid := cookie.Value

	q := `SELECT authorizationid FROM users WHERE id=$1`
	var authID sql.NullInt64
	if err = db.Get(&authID, q, uid); err != nil {
		writeErrorInternal(w, err)
		return
	}
	if !authID.Valid {
		// We don't have an auth request in the database for this user,
		// which is fine.
		goto Response
	}
	q = `UPDATE authorizations SET authorizedat=$1 WHERE id=$2`
	_, err = db.Exec(q, time.Now(), authID)
	if err != nil {
		writeErrorInternal(w, err)
		return
	}
Response:
	w.WriteHeader(http.StatusOK)
}

// HAPIForgotPasswordSubmit asks the server to send the user a "Forgot
// Password" email with instructions for resetting their password.
func HAPIForgotPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, err)
		return
	}

	var user dt.User
	q := `SELECT id, name, email FROM users WHERE email=$1`
	err := db.Get(&user, q, r.FormValue("Email"))
	if err == sql.ErrNoRows {
		writeError(w, errors.New("Sorry, there's no record of that email. Are you sure that's the email you used to sign up with and that you typed it correctly?"))
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	secret := RandSeq(40)
	q = `INSERT INTO passwordresets (userid, secret) VALUES ($1, $2)`
	if _, err = db.Exec(q, user.ID, secret); err != nil {
		writeError(w, err)
		return
	}
	if len(emailsender.Drivers()) == 0 {
		writeError(w, errors.New("Sorry, this feature is not enabled. To be enabled, an email driver must be imported."))
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HAPIResetPasswordSubmit is arrived at through the email generated by
// HAPIForgotPasswordSubmit. This endpoint resets the user password with
// another bcrypt hash after validating on the server that their new password is
// sufficient.
func HAPIResetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeError(w, err)
		return
	}
	if len(r.FormValue("Password")) < 8 {
		writeError(w, errors.New("Your password must be at least 8 characters"))
		return
	}

	var uid uint64
	q := `SELECT userid FROM passwordresets
	      WHERE secret=$1 AND
	      createdat >= CURRENT_TIMESTAMP - interval '30 minutes'`
	err := db.Get(&uid, q, r.FormValue("Secret"))
	if err == sql.ErrNoRows {
		writeError(w, errors.New("Sorry, that information doesn't match our records."))
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}

	hpw, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("Password")), 10)
	if err != nil {
		writeError(w, err)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, err)
		return
	}

	q = `UPDATE users SET password=$1 WHERE id=$2`
	if _, err = tx.Exec(q, hpw, uid); err != nil {
		writeError(w, err)
		return
	}
	q = `DELETE FROM passwordresets WHERE secret=$1`
	if _, err = tx.Exec(q, r.FormValue("Secret")); err != nil {
		writeError(w, err)
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HAPIPlugins responds with all of the server's installed plugin
// configurations from each their respective plugin.json files.
func HAPIPlugins(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ABOT_ENV") != "test" {
		if !Admin(w, r) {
			return
		}
		if !LoggedIn(w, r) {
			return
		}
	}
	// Read plugins.json, unmarshal into struct
	contents, err := ioutil.ReadFile("./plugins.json")
	if err != nil {
		writeError(w, err)
		return
	}
	var plugins PluginJSON
	if err = json.Unmarshal(contents, &plugins); err != nil {
		writeError(w, err)
		return
	}

	var pJSON struct {
		Plugins []json.RawMessage
	}
	for url := range plugins.Dependencies {
		// Add each plugin.json to array of plugins
		p := filepath.Join(os.Getenv("GOPATH"), "src", url,
			"plugin.json")
		var byt []byte
		byt, err = ioutil.ReadFile(p)
		if err != nil {
			writeError(w, err)
			return
		}
		pJSON.Plugins = append(pJSON.Plugins, byt)
	}
	writeBytes(w, pJSON)
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
		return nil, "", err
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
func initCMDGroup(router *httprouter.Router) {
	cmdch := make(chan string, 10)
	addconnch := make(chan *cmdConn, 10)
	delconnch := make(chan *cmdConn, 10)

	go cmder(cmdch, addconnch, delconnch)

	router.GET("/cmd/reload", func(w http.ResponseWriter, r *http.Request,
		ps httprouter.Params) {
		cmdch <- "reload"
		w.WriteHeader(http.StatusOK)
	})
	router.Handler("GET", "/ws", w.Handler(func(ws *w.Conn) {
		respch := make(chan bool)
		conn := &cmdConn{ws: ws, respch: respch}
		addconnch <- conn
		<-respch
		delconnch <- conn
	}))
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
func LoggedIn(w http.ResponseWriter, r *http.Request) bool {
	log.Debug("validating logged in")

	w.Header().Set("WWW-Authenticate", bearerAuthKey+" realm=Restricted")
	auth := r.Header.Get("Authorization")
	l := len(bearerAuthKey)

	// Ensure client sent the token
	if len(auth) <= l+1 || auth[:l] != bearerAuthKey {
		log.Debug("client did not send token")
		writeErrorAuth(w, errors.New("missing Bearer token"))
		return false
	}

	// Ensure the token is still valid
	cookie, err := r.Cookie("issuedAt")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	issuedAt, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	t := time.Unix(issuedAt, 0)
	if t.Add(72 * time.Hour).Before(time.Now()) {
		log.Debug("token expired")
		writeErrorAuth(w, errors.New("missing Bearer token"))
		return false
	}

	// Ensure the token has not been tampered with
	b, err := base64.StdEncoding.DecodeString(auth[l+1:])
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	cookie, err = r.Cookie("scopes")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	scopes := strings.Fields(cookie.Value)
	cookie, err = r.Cookie("id")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	userID, err := strconv.ParseUint(cookie.Value, 10, 64)
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	cookie, err = r.Cookie("email")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	a := Header{
		ID:       userID,
		Email:    cookie.Value,
		Scopes:   scopes,
		IssuedAt: issuedAt,
	}
	byt, err := json.Marshal(a)
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	known := hmac.New(sha512.New, []byte(os.Getenv("ABOT_SECRET")))
	_, err = known.Write(byt)
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	ok := hmac.Equal(known.Sum(nil), b)
	if !ok {
		log.Info("token tampered for user", userID)
		writeErrorAuth(w, errors.New("Bearer token tampered"))
		return false
	}
	log.Debug("validated logged in")
	return true
}

// CSRF ensures that any forms posted to Abot are protected against Cross-Site
// Request Forgery. Without this function, Abot would be vulnerable to the
// attack because tokens are stored client-side in cookies.
func CSRF(w http.ResponseWriter, r *http.Request) bool {
	// TODO look into other session-based temporary storage systems for
	// these csrf tokens to prevent hitting the database.  Whatever is
	// selected must *not* introduce an external (system) dependency like
	// memcached/Redis. Bolt might be an option.
	log.Debug("validating csrf")
	var label string
	q := `SELECT label FROM sessions
	      WHERE userid=$1 AND label='csrfToken' AND token=$2`
	cookie, err := r.Cookie("id")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	uid := cookie.Value
	cookie, err = r.Cookie("csrfToken")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	err = db.Get(&label, q, uid, cookie.Value)
	if err == sql.ErrNoRows {
		writeErrorAuth(w, errors.New("invalid CSRF token"))
		return false
	}
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	log.Debug("validated csrf")
	return true
}

// Admin ensures that the current user is an admin. We trust the scopes
// presented by the client because they're validated through HMAC in LoggedIn().
func Admin(w http.ResponseWriter, r *http.Request) bool {
	log.Debug("validating admin")
	cookie, err := r.Cookie("scopes")
	if err != nil {
		writeErrorInternal(w, err)
		return false
	}
	scopes := strings.Fields(cookie.Value)
	for _, scope := range scopes {
		if scope == "admin" {
			log.Debug("validated admin")
			return true
		}
	}
	writeErrorAuth(w, errors.New("user is not an admin"))
	return false
}

func writeBytes(w http.ResponseWriter, x interface{}) {
	byt, err := json.Marshal(x)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(byt); err != nil {
		writeError(w, err)
	}
}

func writeErrorInternal(w http.ResponseWriter, err error) {
	log.Info("failed", err)
	w.WriteHeader(http.StatusInternalServerError)
	writeError(w, err)
}

func writeErrorAuth(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	writeError(w, err)
}

func writeError(w http.ResponseWriter, err error) {
	tmp := strings.Replace(err.Error(), `"`, "'", -1)
	errS := struct{ Msg string }{Msg: tmp}
	byt, err := json.Marshal(errS)
	if err != nil {
		log.Info("failed to marshal error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(byt); err != nil {
		log.Info("failed to write error", err)
	}
}
