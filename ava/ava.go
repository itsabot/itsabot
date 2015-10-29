package main

import (
	"bytes"
	"database/sql"
	"errors"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jbrukh/bayesian"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo"
	mw "github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo/middleware"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/Godeps/_workspace/src/golang.org/x/crypto/bcrypt"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

var db *sqlx.DB
var bayes *bayesian.Classifier
var ErrInvalidCommand = errors.New("invalid command")
var ErrMissingPackage = errors.New("missing package")

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	rand.Seed(time.Now().UnixNano())
	app := cli.NewApp()
	app.Name = "ava"
	app.Usage = "general purpose ai platform"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "server, s",
			Usage: "run server",
		},
		cli.StringFlag{
			Name:  "port, p",
			Usage: "set port for server",
		},
		cli.BoolFlag{
			Name:  "install, i",
			Usage: "install packages in package.json",
		},
	}
	app.Action = func(c *cli.Context) {
		showHelp := true
		if c.Bool("install") {
			log.Println("TODO: install packages")
			showHelp = false
		}
		if c.Bool("server") {
			db = connectDB()
			startServer(os.Getenv("PORT"))
			showHelp = false
		}
		if showHelp {
			cli.ShowAppHelp(c)
		}
	}
	app.Run(os.Args)
}

func startServer(port string) {
	var err error
	if err = checkRequiredEnvVars(); err != nil {
		log.Println("err:", err)
	}
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Println("err: loading classifier:", err)
	}
	log.Println("booting local server")
	bootRPCServer(port)
	bootTwilio()
	bootDependencies()
	e := echo.New()
	initRoutes(e)
	log.Println("booted ava")
	e.Run(":" + port)
}

func bootRPCServer(port string) {
	ava := new(Ava)
	if err := rpc.Register(ava); err != nil {
		log.Println("register ava in rpc", err)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		log.Println("convert port to int", err)
	}
	pt := strconv.Itoa(p + 1)
	l, err := net.Listen("tcp", ":"+pt)
	log.Println("booting rpc server", pt)
	if err != nil {
		log.Println("err: rpc listen: ", err)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("err: rpc accept: ", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
}

func connectDB() *sqlx.DB {
	log.Println("connecting to db")
	var d *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		d, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		d, err = sqlx.Connect("postgres",
			"user=egtann dbname=ava sslmode=disable")
	}
	if err != nil {
		log.Println("could not connect to db ", err.Error())
	}
	return d
}

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger(), mw.Gzip(), mw.Recover())
	e.SetDebug(true)

	e.Static("/public/css", "assets/css")
	e.Static("/public/js", "assets/js")
	e.Static("/public/images", "assets/images")

	// Web routes
	e.Get("/", handlerIndex)
	e.Get("/signup", handlerSignup)
	e.Post("/signup", handlerSignupSubmit)
	e.Get("/login", handlerLogin)
	e.Post("/login", handlerLoginSubmit)
	e.Get("/success", handlerLoginSuccess)
	e.Get("/train", handlerTrain)

	// API routes
	e.Post("/", handlerMain)
	e.Post("/twilio", handlerTwilio)
	e.Get("/api/sentence.json", handlerSentence)
	e.Put("/api/sentence.json", handlerTrainSentence)
}

func handlerIndex(c *echo.Context) error {
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		log.Fatalln(err)
	}
	tmplIndex, err := template.ParseFiles("assets/html/index.html")
	if err != nil {
		log.Fatalln(err)
	}
	var s []byte
	b := bytes.NewBuffer(s)
	if err := tmplIndex.Execute(b, struct{}{}); err != nil {
		log.Fatalln(err)
	}
	b2 := bytes.NewBuffer(s)
	if err := tmplLayout.Execute(b2, b); err != nil {
		log.Fatalln(err)
	}
	if err = c.HTML(http.StatusOK, "%s", b2); err != nil {
		return err
	}
	return nil
}

func handlerTrain(c *echo.Context) error {
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		log.Fatalln(err)
	}
	tmplIndex, err := template.ParseFiles("assets/html/train.html")
	if err != nil {
		log.Fatalln(err)
	}
	var s []byte
	b := bytes.NewBuffer(s)
	if err := tmplIndex.Execute(b, struct{}{}); err != nil {
		log.Fatalln(err)
	}
	b2 := bytes.NewBuffer(s)
	if err := tmplLayout.Execute(b2, b); err != nil {
		log.Fatalln(err)
	}
	if err = c.HTML(http.StatusOK, "%s", b2); err != nil {
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

func handlerSentence(c *echo.Context) error {
	var q string
	var sent struct {
		Id       int
		Sentence string
	}
	if len(c.Query("id")) > 0 {
		q = `
		SELECT id, sentence FROM trainings
		WHERE trained=FALSE AND id=$1
		ORDER BY createdat DESC`
		err := db.Get(&sent, q, c.Query("id"))
		if err != nil && err != sql.ErrNoRows {
			return err
		}
	} else {
		q = `
		SELECT id, sentence FROM trainings
		WHERE trained=FALSE
		ORDER BY createdat DESC`
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

func handlerTrainSentence(c *echo.Context) error {
	var data struct {
		Id       int
		Sentence string
	}
	if err := c.Bind(&data); err != nil {
		return err
	}
	if err := train(bayes, data.Sentence); err != nil {
		return err
	}
	q := `UPDATE trainings SET trained=TRUE WHERE id=$1`
	res, err := db.Exec(q, data.Id)
	if err != nil {
		return err
	}
	num, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if num == 0 {
		return sql.ErrNoRows
	}
	if err = c.JSON(http.StatusOK, nil); err != nil {
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

func processText(c *echo.Context) (string, error) {
	cmd := c.Get("cmd").(string)
	if len(cmd) == 0 {
		return "", ErrInvalidCommand
	}
	if len(cmd) >= 5 && strings.ToLower(cmd)[0:5] == "train" {
		if err := train(bayes, cmd[6:]); err != nil {
			return "", err
		}
		return "", nil
	}
	si, needsTraining, err := classify(bayes, cmd)
	if err != nil {
		log.Println("classifying sentence ", err)
	}
	uid, fid, fidT := validateParams(c)
	in := &datatypes.Input{
		Sentence:        cmd,
		StructuredInput: si,
		UserID:          uid,
		FlexID:          fid,
		FlexIDType:      fidT,
	}
	u, err := getUser(in)
	if err == ErrMissingUser {
		log.Println(err)
	} else if err != nil {
		log.Println("getUser: ", err)
		return "", err
	}
	m := &datatypes.Message{User: u, Input: in}
	m, ctxAdded, err := addContext(m)
	if err != nil {
		log.Println("addContext: ", err)
	}
	ret, pname, route, err := callPkg(m, ctxAdded)
	if err != nil && err != ErrMissingPackage {
		return "", err
	}
	if len(ret.Sentence) == 0 {
		ret.Sentence = language.Confused()
	}
	in.StructuredInput = si
	err = saveStructuredInput(m, ret.ResponseID, pname, route)
	if err != nil {
		return ret.Sentence, err
	}
	if needsTraining {
		log.Println("needed training")
		if err = supervisedTrain(in.Sentence); err != nil {
			return ret.Sentence, err
		}
	}
	return ret.Sentence, nil
}

func sendMessage(to, msg string) error {
	params := twilio.MessageParams{Body: msg}
	_, resp, err := tc.Messages.Send("+14242971568", to, params)
	log.Println(resp)
	if err != nil {
		return err
	}
	return nil
}

func handlerLogin(c *echo.Context) error {
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		log.Fatalln(err)
	}
	tmplLogin, err := template.ParseFiles("assets/html/login.html")
	if err != nil {
		log.Fatalln(err)
	}
	var s []byte
	b := bytes.NewBuffer(s)
	var data struct{ Error string }
	if c.Get("err") != nil {
		data.Error = c.Get("err").(error).Error()
		c.Set("err", nil)
	}
	if err := tmplLogin.Execute(b, data); err != nil {
		log.Fatalln(err)
	}
	b2 := bytes.NewBuffer(s)
	if err := tmplLayout.Execute(b2, b); err != nil {
		log.Fatalln(err)
	}
	if err = c.HTML(http.StatusOK, "%s", b2); err != nil {
		return err
	}
	return nil
}

func handlerSignup(c *echo.Context) error {
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		return err
	}
	tmplSignup, err := template.ParseFiles("assets/html/signup.html")
	if err != nil {
		return err
	}
	var s []byte
	b := bytes.NewBuffer(s)
	data := struct{ Error string }{}
	if c.Get("err") != nil {
		data.Error = c.Get("err").(error).Error()
		c.Set("err", nil)
	}
	if err := tmplSignup.Execute(b, data); err != nil {
		return err
	}
	b2 := bytes.NewBuffer(s)
	if err := tmplLayout.Execute(b2, b); err != nil {
		return err
	}
	if err = c.HTML(http.StatusOK, "%s", b2); err != nil {
		return err
	}
	return nil
}

func handlerLoginSubmit(c *echo.Context) error {
	var u struct {
		Id       int
		Password []byte
	}
	var err error
	q := `SELECT id, password FROM users WHERE email=$1`
	err = db.Get(&u, q, c.Form("email"))
	if err == sql.ErrNoRows {
		err = errors.New("Invalid username/password combination")
		goto Response
	} else if err != nil {
		goto Response
	}
	err = bcrypt.CompareHashAndPassword(u.Password, []byte(c.Form("pw")))
	if err != nil {
		goto Response
	}
Response:
	if err != nil {
		c.Set("err", err)
		return handlerLogin(c)
	}
	return handlerLoginSuccess(c)
}

func handlerLoginSuccess(c *echo.Context) error {
	tmplLayout, err := template.ParseFiles("assets/html/layout.html")
	if err != nil {
		return err
	}
	tmplSignup, err := template.ParseFiles("assets/html/loginsuccess.html")
	if err != nil {
		return err
	}
	var s []byte
	b := bytes.NewBuffer(s)
	if err = tmplSignup.Execute(b, struct{}{}); err != nil {
		return err
	}
	b2 := bytes.NewBuffer(s)
	if err = tmplLayout.Execute(b2, b); err != nil {
		return err
	}
	to := c.Form("flexid")
	err = sendMessage(to, "Thanks for signing in! How can I help you?")
	if err != nil {
		return err
	}
	if err = c.HTML(http.StatusOK, "%s", b2); err != nil {
		return err
	}
	return nil
}

func handlerSignupSubmit(c *echo.Context) error {
	name := c.Form("name")
	email := c.Form("email")
	pw := c.Form("pw")
	fid := c.Form("flexid")
	fidT := c.Form("flexidtype")
	var err error
	var hpw []byte
	var q string
	var uid int
	var tx *sqlx.Tx
	if len(name) == 0 {
		err = errors.New("You must enter a name.")
		goto Response
	}
	if len(email) == 0 || !strings.ContainsAny(email, "@") {
		err = errors.New("You must enter an email.")
		goto Response
	}
	if len(pw) < 8 {
		err = errors.New("Your password must be at least 8 characters.")
		goto Response
	}
	hpw, err = bcrypt.GenerateFromPassword([]byte(pw), 10)
	if err != nil {
		goto Response
	}
	tx, err = db.Beginx()
	if err != nil {
		err = errors.New("Something went wrong. Try again.")
		goto Response
	}
	q = `INSERT INTO users (name, email, password, locationid)
	     VALUES ($1, $2, $3, 0)
	     RETURNING id`
	err = tx.QueryRowx(q, name, email, hpw).Scan(&uid)
	if err != nil && err.Error() ==
		`pq: duplicate key value violates unique constraint "users_email_key"` {
		err = errors.New("Sorry, that email is taken.")
	}
	q = `INSERT INTO userflexids (userid, flexid, flexidtype)
	     VALUES ($1, $2, $3)`
	_, err = tx.Exec(q, uid, fid, fidT)
	if err != nil {
		log.Println("uid", uid, "fid", fid, "fidT", fidT)
		err = errors.New(
			"Couldn't register you. Did you use the link sent to you?")
		goto Response
	}
	if err = tx.Commit(); err != nil {
		err = errors.New("Something went wrong. Please try again.")
		goto Response
	}
Response:
	if err != nil {
		c.Set("err", err)
		return handlerSignup(c)
	}
	return handlerLoginSuccess(c)
}

func validateParams(c *echo.Context) (int, string, int) {
	var uid, fidT int
	var fid string
	var err error
	var tmp interface{}
	tmp = c.Get("uid")
	if tmp != nil {
		uid, err = strconv.Atoi(tmp.(string))
		if err.Error() == `strconv.ParseInt: parsing "": invalid syntax` {
			uid = 0
		} else if err != nil {
			log.Fatalln(err)
		}
	}
	tmp = c.Get("flexid")
	if tmp != nil {
		fid = tmp.(string)
		if len(fid) == 0 {
			log.Fatalln("flexid is blank")
		}
	}
	tmp = c.Get("flexidtype")
	if tmp != nil {
		var ok bool
		fidT, ok = tmp.(int)
		if !ok {
			fidT, err = strconv.Atoi(tmp.(string))
			if err != nil && err.Error() ==
				`strconv.ParseInt: parsing "": invalid syntax` {
				fidT = 0
			} else if err != nil {
				log.Fatalln(err)
			}
		}
		if fidT == 0 {
			log.Fatalln("flexidtype cannot be 0")
		}
	}
	return uid, fid, fidT
}

func checkRequiredEnvVars() error {
	port := os.Getenv("PORT")
	_, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("PORT is not set to an integer")
	}
	base := os.Getenv("BASE_URL")
	l := len(base)
	if l == 0 {
		return errors.New("BASE_URL not set")
	}
	if l < 4 || base[0:4] != "http" {
		return errors.New("BASE_URL invalid. Must include http/https")
	}
	// TODO Check for DATABASE_URL if AVA_ENV==production
	return nil
}
