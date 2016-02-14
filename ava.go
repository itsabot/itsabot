package main

import (
	"errors"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"regexp"
	"strconv"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo"
	_ "github.com/avabot/ava/Godeps/_workspace/src/github.com/lib/pq"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/stripe/stripe-go"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/subosito/twilio"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/nlp"
	"github.com/avabot/ava/shared/sms"
)

// TODO variable routes. e.g. "Help me get drunk" could route to purchase
// (alcohol) or bars nearby. Ava should ask the user which route to send them
// to on packages with overlapping routes.

var db *sqlx.DB
var tc *twilio.Client
var mc *dt.MailClient
var ws dt.AtomicWebSocketMap = dt.NewAtomicWebSocketMap()
var ner nlp.Classifier
var offensive map[string]struct{}
var phoneRegex = regexp.MustCompile(`^\+?[0-9\-\s()]+$`)
var (
	ErrInvalidCommand    = errors.New("invalid command")
	ErrMissingPackage    = errors.New("missing package")
	ErrInvalidUserPass   = errors.New("Invalid username/password combination")
	ErrMissingFlexIdType = errors.New("missing flexidtype")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(log.DebugLevel)
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
			log.Info("TODO: install packages")
			showHelp = false
		}
		if c.Bool("server") {
			db = connectDB()
			startServer()
			showHelp = false
		}
		if showHelp {
			cli.ShowAppHelp(c)
		}
	}
	app.Run(os.Args)
}

func startServer() {
	if err := checkRequiredEnvVars(); err != nil {
		log.Errorln("checking env vars", err)
	}
	addr, err := bootRPCServer()
	if err != nil {
		log.Fatalln("unable to boot rpc server:", err)
	}
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")
	tc = sms.NewClient()
	mc = dt.NewMailClient()
	appVocab = dt.NewAtomicMap()
	go bootDependencies(addr)
	ner, err = buildClassifier()
	if err != nil {
		log.Errorln("loading classifier", err)
	}
	offensive, err = buildOffensiveMap()
	if err != nil {
		log.Errorln("building offensive map", err)
	}
	log.Infoln("booting ava http server")
	e := echo.New()
	initRoutes(e)
	e.Run(":" + os.Getenv("PORT"))
}

// bootRPCServer starts the rpc for ava core in a go routine and returns
// the server address
func bootRPCServer() (addr string, err error) {
	log.Debugln("booting ava core rpc server")
	ava := new(Ava)
	if err = rpc.Register(ava); err != nil {
		return
	}
	var ln net.Listener
	if ln, err = net.Listen("tcp", ":0"); err != nil {
		return
	}
	addr = ln.Addr().String()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Errorln("rpc accept", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
	return // using named return params
}

func connectDB() *sqlx.DB {
	log.Debugln("connecting to db")
	var d *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		d, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		d, err = sqlx.Connect("postgres",
			"user=postgres dbname=ava sslmode=disable")
	}
	if err != nil {
		log.Errorln("connecting to db", err)
	}
	log.Infoln("connected to db")
	return d
}

func preprocess(c *echo.Context) (*dt.Msg, error) {
	cmd := c.Get("cmd").(string)
	if len(cmd) == 0 {
		return nil, ErrInvalidCommand
	}
	uid, fid, fidT := validateParams(c)
	u, err := getUser(uid, fid, flexIDType(fidT))
	if err != nil {
		return nil, err
	}
	msg := dt.NewMsg(db, ner, u, cmd)
	if err = msg.Update(db); err != nil {
		return nil, err
	}
	// TODO trigger training if needed (see buildInput)
	return msg, nil
}

func processText(c *echo.Context) (ret string, uid uint64, err error) {
	msg, err := preprocess(c)
	if err != nil {
		log.WithField("fn", "preprocessForMessage").Error(err)
		return "", 0, err
	}
	log.Debugln("processed input into message...")
	log.Debugln("commands:", msg.StructuredInput.Commands)
	log.Debugln(" objects:", msg.StructuredInput.Objects)
	log.Debugln("  people:", msg.StructuredInput.People)
	pkg, route, followup, err := getPkg(msg)
	if err != nil {
		log.WithField("fn", "getPkg").Error(err)
		return "", msg.User.ID, err
	}
	msg.Route = route
	if pkg == nil {
		msg.Package = ""
	} else {
		msg.Package = pkg.P.Config.Name
	}
	if err = msg.Save(db); err != nil {
		return "", msg.User.ID, err
	}
	ret = respondWithOffense(offensive, msg)
	if len(ret) == 0 {
		log.Debugln("followup?", followup)
		ret, err = callPkg(pkg, msg, followup)
		if err != nil {
			return "", msg.User.ID, err
		}
		responseNeeded := true
		if len(ret) == 0 {
			responseNeeded, ret = respondWithNicety(msg)
		}
		if !responseNeeded {
			return "", msg.User.ID, nil
		}
	}
	log.Debugln("here...", ret)
	m := &dt.Msg{}
	m.AvaSent = true
	m.User = msg.User
	if len(ret) == 0 {
		m.Sentence = language.Confused()
		msg.NeedsTraining = true
		if err = msg.Update(db); err != nil {
			return "", m.User.ID, err
		}
	} else {
		m.Sentence = ret
	}
	if pkg != nil {
		m.Package = pkg.P.Config.Name
	}
	if err = m.Save(db); err != nil {
		return "", m.User.ID, err
	}
	/*
		// TODO handle earlier when classifying
		if ctx.NeedsTraining {
			log.WithField("inputID", id).Infoln("needed training")
			if err = supervisedTrain(ctx.Msg); err != nil {
				return ret.Sentence, err
			}
		}
	*/
	return m.Sentence, m.User.ID, nil
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
		if err != nil && err.Error() == `strconv.ParseInt: parsing "": invalid syntax` {
			uid = 0
		} else if err != nil {
			log.WithField("fn", "validateParams").Fatalln(err)
		}
	}
	if uid > 0 {
		return uid, "", 0
	}
	tmp, ok = c.Get("flexid").(string)
	if !ok {
		tmp = ""
	}
	if len(tmp) > 0 {
		fid = tmp
		if len(fid) == 0 {
			log.WithField("fn", "validateParams").
				Fatalln("flexid is blank")
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
			// default to 2 (SMS)
			fidT = 2
		} else if err != nil {
			log.WithField("fn", "validateParams").Fatalln(err)
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
