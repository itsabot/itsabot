package main

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jbrukh/bayesian"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/sms"
)

// TODO variable routes. e.g. "Help me get drunk" could route to purchase
// (alcohol) or bars nearby. Ava should ask the user which route to send them
// to on packages with overlapping routes.

var db *sqlx.DB
var tc *twilio.Client
var bayes *bayesian.Classifier
var phoneRegex *regexp.Regexp
var ErrInvalidCommand = errors.New("invalid command")
var ErrMissingPackage = errors.New("missing package")
var ErrInvalidUserPass = errors.New("Invalid username/password combination")

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
	phoneRegex = regexp.MustCompile(`^\+?[0-9\-\s()]+$`)
	if err = checkRequiredEnvVars(); err != nil {
		log.Println("err:", err)
	}
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Println("err: loading classifier:", err)
	}
	log.Println("booting local server")
	bootRPCServer(port)
	tc = sms.NewClient()
	bootDependencies()
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")
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
	si, annotated, needsTraining, err := classify(bayes, cmd)
	if err != nil {
		log.Println("classifying sentence ", err)
	}
	uid, fid, fidT := validateParams(c)
	in := &dt.Input{
		Sentence:          cmd,
		StructuredInput:   si,
		UserID:            uid,
		FlexID:            fid,
		FlexIDType:        fidT,
		SentenceAnnotated: annotated,
	}
	u, err := getUser(in)
	if err == ErrMissingUser {
		log.Println(err)
	} else if err != nil {
		log.Println("getUser: ", err)
		return "", err
	}
	m := &dt.Msg{User: u, Input: in}
	m, ctxAdded, err := addContext(m)
	if err != nil {
		log.Println("addContext: ", err)
	}
	ret, pname, route, err := callPkg(m, ctxAdded)
	if err != nil && err != ErrMissingPackage {
		return "", err
	}
	if len(ret.Sentence) == 0 {
		inAuth, err := checkActiveAuthorization(m)
		if err != nil {
			return "", err
		}
		if !inAuth {
			ret.Sentence = language.Confused()
		}
	}
	in.StructuredInput = si
	id, err := saveStructuredInput(m, ret.ResponseID, pname, route)
	if err != nil {
		return ret.Sentence, err
	}
	in.ID = id
	if needsTraining {
		log.Println("needed training")
		if err = supervisedTrain(in); err != nil {
			return ret.Sentence, err
		}
	}
	return ret.Sentence, nil
}

func validateParams(c *echo.Context) (uint64, string, int) {
	var uid uint64
	var fidT int
	var fid string
	var err error
	tmp, ok := c.Get("uid").(string)
	if !ok {
		tmp = ""
	}
	if len(tmp) > 0 {
		uid, err = strconv.ParseUint(tmp, 10, 64)
		if err.Error() == `strconv.ParseInt: parsing "": invalid syntax` {
			uid = 0
		} else if err != nil {
			log.Fatalln(err)
		}
	}
	tmp, ok = c.Get("flexid").(string)
	if !ok {
		tmp = ""
	}
	if len(tmp) > 0 {
		fid = tmp
		if len(fid) == 0 {
			log.Fatalln("flexid is blank")
		}
	}
	tmp, ok = c.Get("flexidtype").(string)
	if !ok {
		tmp = ""
	}
	if len(tmp) > 0 {
		fidT, err = strconv.Atoi(tmp)
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
