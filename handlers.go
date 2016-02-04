package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo"
	mw "github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo/middleware"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/satori/go.uuid"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go/card"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go/customer"
	"github.com/avabot/ava/Godeps/_workspace/src/golang.org/x/crypto/bcrypt"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/sms"
)

var letters = []rune(
	"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger(), mw.Gzip(), mw.Recover())
	e.SetDebug(true)

	e.Static("/public/css", "assets/css")
	e.Static("/public/js", "public/js")
	e.Static("/public/images", "assets/images")

	// Web routes
	e.Get("/", handlerIndex)

	// API routes
	e.Post("/", handlerMain)
	e.Post("/twilio", handlerTwilio)
	e.Get("/api/profile.json", handlerAPIProfile)
	e.Put("/api/profile.json", handlerAPIProfileView)
	e.Post("/api/login.json", handlerAPILoginSubmit)
	e.Post("/api/signup.json", handlerAPISignupSubmit)
	e.Post("/api/forgot_password.json", handlerAPIForgotPasswordSubmit)
	e.Post("/api/reset_password.json", handlerAPIResetPasswordSubmit)
	e.Post("/api/cards.json", handlerAPICardSubmit)
	e.Delete("/api/cards.json", handlerAPICardDelete)
	e.Get("/api/conversation.json", handlerAPIConversationsShow)
	e.Post("/api/conversations.json", handlerAPIConversationsCreate)
	e.Get("/api/conversations.json", handlerAPIConversations)
	e.Post("/api/conversations_preview.json", handlerAPIPreviewCmd)
}

func handlerIndex(c *echo.Context) error {
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		log.Fatalln(err)
	}
	var s []byte
	b := bytes.NewBuffer(s)
	data := struct {
		StripeKey string
	}{StripeKey: os.Getenv("STRIPE_PUBLIC_KEY")}
	if err := tmplLayout.Execute(b, data); err != nil {
		log.Fatalln(err)
	}
	if err = c.HTML(http.StatusOK, string(b.Bytes())); err != nil {
		return err
	}
	return nil
}

func handlerTwilio(c *echo.Context) error {
	c.Set("cmd", c.Form("Body"))
	c.Set("flexid", c.Form("From"))
	c.Set("flexidtype", 2)
	ret, err := processText(c)
	if err != nil {
		return err
	}
	var resp twilioResp
	if len(ret) == 0 {
		resp = twilioResp{}
	} else {
		resp = twilioResp{Message: ret}
	}
	if err = c.XML(http.StatusOK, resp); err != nil {
		return err
	}
	return nil
}

func handlerAPISentence(c *echo.Context) error {
	var q string
	var sent struct {
		ID             int
		ForeignID      string
		Sentence       string
		MaxAssignments int
	}
	if len(c.Query("id")) > 0 {
		q = `
		SELECT id, foreignid, sentence, maxassignments FROM trainings
		WHERE trainedcount<3 AND id=$1
		OFFSET FLOOR(RANDOM() * (SELECT COUNT(*) FROM trainings WHERE trainedcount<3 AND id=$1))`
		err := db.Get(&sent, q, c.Query("id"))
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	} else {
		q = `
		SELECT id, foreignid, sentence, maxassignments FROM trainings
		WHERE trainedcount<3
		OFFSET FLOOR(RANDOM() * (SELECT COUNT(*) FROM trainings WHERE trainedcount<3))`
		err := db.Get(&sent, q)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	}
	if err := c.JSON(http.StatusOK, sent); err != nil {
		return err
	}
	return nil
}

func handlerMain(c *echo.Context) error {
	c.Set("cmd", c.Form("cmd"))
	c.Set("flexid", c.Form("flexid"))
	c.Set("flexidtype", c.Form("flexidtype"))
	c.Set("uid", c.Form("uid"))
	ret, err := processText(c)

	if err != nil {
		return err
	}
	if err = c.HTML(http.StatusOK, ret); err != nil {
		return err
	}
	return nil
}

func handlerAPILoginSubmit(c *echo.Context) error {
	var u struct {
		Id       int
		Password []byte
	}
	var req struct {
		Email    string
		Password string
	}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}
	q := `SELECT id, password FROM users WHERE email=$1`
	err := db.Get(&u, q, req.Email)
	if err == sql.ErrNoRows {
		return jsonError(ErrInvalidUserPass)
	} else if err != nil {
		return jsonError(err)
	}
	if u.Id == 0 {
		return jsonError(ErrInvalidUserPass)
	}
	err = bcrypt.CompareHashAndPassword(u.Password, []byte(req.Password))
	if err == bcrypt.ErrMismatchedHashAndPassword ||
		err == bcrypt.ErrHashTooShort {
		return jsonError(ErrInvalidUserPass)
	} else if err != nil {
		return jsonError(err)
	}
	var resp struct {
		Id           int
		SessionToken string
	}
	resp.Id = u.Id
	tmp := uuid.NewV4().Bytes()
	resp.SessionToken = base64.StdEncoding.EncodeToString(tmp)
	// TODO save session token
	if err = c.JSON(http.StatusOK, resp); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPISignupSubmit(c *echo.Context) error {
	req := struct {
		Name     string
		Email    string
		Password string
		FID      string
	}{}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}

	// validate the request parameters
	if len(req.Name) == 0 {
		return jsonError(errors.New("You must enter a name."))
	}
	if len(req.Email) == 0 || !strings.ContainsAny(req.Email, "@") ||
		!strings.ContainsAny(req.Email, ".") {
		return jsonError(errors.New("You must enter a valid email."))
	}
	if len(req.Password) < 8 {
		return jsonError(errors.New(
			"Your password must be at least 8 characters."))
	}
	if err := validatePhone(req.FID); err != nil {
		return jsonError(err)
	}

	// create the password hash
	//   TODO format phone number for Twilio (international format)
	hpw, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return jsonError(err)
	}

	// XXX temporary to allow dev work w/out stripe
	var stripeCustomerID string
	if os.Getenv("AVA_ENV") == "production" {
		customerParams := &stripe.CustomerParams{Email: req.Email}
		cust, err := customer.New(customerParams)
		if err != nil {
			var js struct {
				Message string
			}
			err = json.Unmarshal([]byte(err.Error()), &js)
			if err != nil {
				return jsonError(err)
			}
			return jsonError(errors.New(js.Message))
		}
		stripeCustomerID = cust.ID
	}

	// db stuff...
	tx, err := db.Beginx()
	if err != nil {
		return jsonError(errors.New("Something went wrong. Try again."))
	}

	q := `INSERT INTO users (name, email, password, locationid, stripecustomerid)
	      VALUES ($1, $2, $3, 0, $4)
	      RETURNING id`
	var uid int
	err = tx.QueryRowx(q, req.Name, req.Email, hpw, stripeCustomerID).Scan(&uid)
	if err != nil && err.Error() ==
		`pq: duplicate key value violates unique constraint "users_email_key"` {
		_ = tx.Rollback()
		return jsonError(errors.New("Sorry, that email is taken."))
	}
	if uid == 0 {
		_ = tx.Rollback()
		return jsonError(errors.New(
			"Something went wrong. Please try again."))
	}

	q = `INSERT INTO userflexids (userid, flexid, flexidtype)
	     VALUES ($1, $2, $3)`
	_, err = tx.Exec(q, uid, req.FID, 2)
	if err != nil {
		_ = tx.Rollback()
		return jsonError(errors.New(
			"Couldn't sign up. Did you use the link sent to you?"))
	}
	if err = tx.Commit(); err != nil {
		return jsonError(errors.New(
			"Something went wrong. Please try again."))
	}
	// end db stuff

	var resp struct {
		Id           int
		SessionToken string
	}
	tmp := uuid.NewV4().Bytes()
	resp.Id = uid
	resp.SessionToken = base64.StdEncoding.EncodeToString(tmp)
	if os.Getenv("AVA_ENV") == "production" {
		fName := strings.Fields(req.Name)[0]
		msg := fmt.Sprintf("Nice to meet you, %s. ", fName)
		msg += "How can I help? Try asking me to help you find a nice bottle of wine."
		if err = sms.SendMessage(tc, req.FID, msg); err != nil {
			return jsonError(err)
		}
	}
	// TODO save session token
	if err = c.JSON(http.StatusOK, resp); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIProfile(c *echo.Context) error {
	uid, err := strconv.Atoi(c.Query("uid"))
	if err != nil {
		return jsonError(err)
	}
	var user struct {
		Name   string
		Email  string
		Phones []dt.Phone
		Cards  []struct {
			Id             int
			CardholderName string
			Last4          string
			ExpMonth       string `db:"expmonth"`
			ExpYear        string `db:"expyear"`
			Brand          string
		}
		Addresses []struct {
			Id      int
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
		return jsonError(err)
	}
	q = `
		SELECT flexid FROM userflexids
		WHERE flexidtype=2 AND userid=$1
		LIMIT 10`
	err = db.Select(&user.Phones, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return jsonError(err)
	}
	q = `
		SELECT id, cardholdername, last4, expmonth, expyear, brand
		FROM cards
		WHERE userid=$1
		LIMIT 10`
	err = db.Select(&user.Cards, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return jsonError(err)
	}
	q = `
		SELECT id, name, line1, line2, city, state, country, zip
		FROM addresses
		WHERE userid=$1
		LIMIT 10`
	err = db.Select(&user.Addresses, q, uid)
	if err != nil && err != sql.ErrNoRows {
		return jsonError(err)
	}
	if err = c.JSON(http.StatusOK, user); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIProfileView(c *echo.Context) error {
	var err error
	req := struct {
		UserID uint64
	}{}
	if err = c.Bind(&req); err != nil {
		return jsonError(err)
	}
	q := `SELECT authorizationid FROM users WHERE id=$1`
	var authID sql.NullInt64
	if err = db.Get(&authID, q, req.UserID); err != nil {
		return jsonError(err)
	}
	if !authID.Valid {
		goto Response
	}
	q = `UPDATE authorizations SET authorizedat=$1 WHERE id=$2`
	_, err = db.Exec(q, time.Now(), authID)
	if err != nil && err != sql.ErrNoRows {
		return jsonError(err)
	}
Response:
	err = c.JSON(http.StatusOK, nil)
	if err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIForgotPasswordSubmit(c *echo.Context) error {
	var req struct {
		Email string
	}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}
	var user dt.User
	q := `SELECT id, name, email FROM users WHERE email=$1`
	err := db.Get(&user, q, req.Email)
	if err == sql.ErrNoRows {
		return jsonError(errors.New("Sorry, there's no record of that email. Are you sure that's the email you used to sign up with and that you typed it correctly?"))
	}
	if err != nil {
		return jsonError(err)
	}
	secret := randSeq(40)
	q = `INSERT INTO passwordresets (userid, secret) VALUES ($1, $2)`
	if _, err := db.Exec(q, user.ID, secret); err != nil {
		return jsonError(err)
	}
	h := `<html><body>`
	h += fmt.Sprintf("<p>Hi %s:</p>", user.Name)
	h += "<p>Please click the following link to reset your password. This link will expire in 30 minutes.</p>"
	h += fmt.Sprintf("<p>%s</p>", os.Getenv("BASE_URL")+"?/reset_password?s="+secret)
	h += "<p>If you received this email in error, please ignore it.</p>"
	h += "<p>Have a great day!</p>"
	h += "<p>-Ava</p>"
	h += "</body></html>"
	if err := mc.Send("Password reset", h, &user); err != nil {
		return jsonError(err)
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIResetPasswordSubmit(c *echo.Context) error {
	var req struct {
		Secret   string
		Password string
	}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}
	if len(req.Password) < 8 {
		return jsonError(errors.New("Your password must be at least 8 characters"))
	}
	userid := uint64(0)
	q := `SELECT userid FROM passwordresets
	      WHERE secret=$1 AND
	            createdat >= CURRENT_TIMESTAMP - interval '30 minutes'`
	err := db.Get(&userid, q, req.Secret)
	if err == sql.ErrNoRows {
		return jsonError(errors.New("Sorry, that information doesn't match our records."))
	}
	if err != nil {
		return jsonError(err)
	}
	hpw, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return jsonError(err)
	}
	tx, err := db.Begin()
	if err != nil {
		return jsonError(err)
	}
	q = `UPDATE users SET password=$1 WHERE id=$2`
	if _, err = tx.Exec(q, hpw, userid); err != nil {
		return jsonError(err)
	}
	q = `DELETE FROM passwordresets WHERE secret=$1`
	if _, err = tx.Exec(q, req.Secret); err != nil {
		return jsonError(err)
	}
	if err = tx.Commit(); err != nil {
		return jsonError(err)
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPICardSubmit(c *echo.Context) error {
	var req struct {
		StripeToken    string
		CardholderName string
		Last4          string
		Brand          string
		ExpMonth       int
		ExpYear        int
		AddressZip     string
		UserID         int
	}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}
	hZip, err := bcrypt.GenerateFromPassword([]byte(req.AddressZip[:5]), 10)
	if err != nil {
		return jsonError(err)
	}
	log.Println("submitting card for user", req.UserID)
	var userStripeID string
	q := `SELECT stripecustomerid FROM users WHERE id=$1`
	if err := db.Get(&userStripeID, q, req.UserID); err != nil {
		log.Println("err with db.Get")
		return jsonError(err)
	}
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")
	cd, err := card.New(&stripe.CardParams{
		Customer: userStripeID,
		Token:    req.StripeToken,
	})
	if err != nil {
		return jsonError(err)
	}
	var crd struct{ ID int }
	q = `
		INSERT INTO cards
		(userid, last4, cardholdername, expmonth, expyear, brand,
			stripeid, zip5hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`
	row := db.QueryRowx(q, req.UserID, req.Last4, req.CardholderName,
		req.ExpMonth, req.ExpYear, req.Brand, cd.ID, hZip)
	err = row.Scan(&crd.ID)
	if err != nil {
		return jsonError(err)
	}
	if err = c.JSON(http.StatusOK, crd); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPICardDelete(c *echo.Context) error {
	var req struct {
		ID     uint64
		UserID uint64
	}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}
	q := `SELECT stripeid FROM cards WHERE id=$1`
	var crd dt.Card
	if err := db.Get(&crd, q, req.ID); err != nil {
		log.Println("couldn't find stripeid", req.ID)
		return jsonError(err)
	}
	q = `DELETE FROM cards WHERE id=$1 AND userid=$2`
	if _, err := db.Exec(q, req.ID, req.UserID); err != nil {
		log.Println("couldn't find card", req.ID, req.UserID)
		return jsonError(err)
	}
	q = `SELECT stripecustomerid FROM users WHERE id=$1`
	var user dt.User
	if err := db.Get(&user, q, req.UserID); err != nil {
		log.Println("couldn't find stripecustomerid", req.UserID)
		return jsonError(err)
	}
	_, err := card.Del(crd.StripeID, &stripe.CardParams{
		Customer: user.StripeCustomerID,
	})
	if err != nil {
		log.Println("couldn't delete stripe card", crd.StripeID,
			user.StripeCustomerID)
		return jsonError(err)
	}
	err = c.JSON(http.StatusOK, nil)
	if err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIConversations(c *echo.Context) error {
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
		return jsonError(err)
	}
	if err = c.JSON(http.StatusOK, msgs); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIConversationsShow(c *echo.Context) error {
	uid, err := strconv.Atoi(c.Query("uid"))
	if err != nil {
		return jsonError(err)
	}
	var ret []struct {
		Sentence  string
		AvaSent   bool
		CreatedAt time.Time
	}
	q := `SELECT sentence, avasent, createdat
	      FROM messages
	      WHERE userid=$1
	      ORDER BY createdat DESC
	      LIMIT 10`
	if err := db.Select(&ret, q, uid); err != nil {
		return jsonError(err)
	}
	var username string
	q = `SELECT name FROM users WHERE id=$1`
	if err := db.Get(&username, q, uid); err != nil {
		return jsonError(err)
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
		return jsonError(err)
	}
	var prefs []string
	for _, p := range prefsTmp {
		prefs = append(prefs, p.Key+": "+
			strings.ToUpper(p.Value[:1])+p.Value[1:])
	}
	resp := struct {
		Username      string
		Conversations []struct {
			Sentence  string
			AvaSent   bool
			CreatedAt time.Time
		}
		Preferences []string
	}{
		Username:      username,
		Conversations: ret,
		Preferences:   prefs,
	}
	if err := c.JSON(http.StatusOK, resp); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIConversationsCreate(c *echo.Context) error {
	var req struct {
		Sentence string
		UserID   uint64
	}
	if err := c.Bind(&req); err != nil {
		return jsonError(err)
	}
	// TODO record the last flextype used and send the user a response via
	// that. e.g. if message was received from a secondary phone number,
	// respond to the user via that secondary phone number. For now, simply
	// get the first flexid available
	var fid string
	q := `SELECT flexid FROM userflexids WHERE flexidtype=2 AND userid=$1`
	if err := db.Get(&fid, q, req.UserID); err != nil {
		return jsonError(err)
	}
	var id uint64
	q = `INSERT INTO messages
	     (userid, sentence, avasent) VALUES ($1, $2, TRUE) RETURNING id`
	err := db.QueryRowx(q, req.UserID, req.Sentence).Scan(&id)
	if err != nil {
		return jsonError(err)
	}
	/*
		if err = sms.SendMessage(tc, fid, req.Sentence); err != nil {
			q = `DELETE FROM messages WHERE id=$1`
			if _, err = db.Exec(q, id); err != nil {
				return jsonError(err)
			}
			return jsonError(err)
		}
	*/
	if err := c.JSON(http.StatusOK, nil); err != nil {
		return jsonError(err)
	}
	return nil
}

func handlerAPIPreviewCmd(c *echo.Context) error {
	msg, err := preprocess(c)
	if err != nil || msg == nil /* trained */ {
		log.WithField("fn", "preprocessForMessage").Error(err)
		return err
	}
	pkg, route, followup, err := getPkg(msg)
	if err != nil && err != ErrMissingPackage {
		log.WithField("fn", "getPkg").Error(err)
		return err
	}
	msg.Route = route
	msg.Package = pkg.P.Config.Name
	if err = msg.Save(db); err != nil {
		return err
	}
	// TODO use ret.Sentence
	_, err = callPkg(pkg, msg, followup)
	if err != nil {
		return err
	}
	return nil
}

func jsonError(err error) error {
	tmp := strings.Replace(err.Error(), `"`, "'", -1)
	return errors.New(`{"Msg":"` + tmp + `"}`)
}

func validatePhone(s string) error {
	if len(s) < 10 || len(s) > 20 ||
		!phoneRegex.MatchString(s) {
		return errors.New(
			"Your phone must be a valid U.S. number with the area code.")
	}
	if len(s) == 11 && s[0] != '1' {
		return errors.New(
			"Sorry, Ava only serves U.S. customers for now.")
	}
	if len(s) == 12 && s[0] == '+' && s[1] != '1' {
		return errors.New(
			"Sorry, Ava only serves U.S. customers for now.")
	}
	return nil
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
