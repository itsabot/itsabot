package main

import (
	"errors"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jbrukh/bayesian"
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
var bayes *bayesian.Classifier
var phoneRegex = regexp.MustCompile(`^\+?[0-9\-\s()]+$`)
var ErrInvalidCommand = errors.New("invalid command")
var ErrMissingPackage = errors.New("missing package")
var ErrInvalidUserPass = errors.New("Invalid username/password combination")

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
	go bootDependencies(addr)

	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Errorln("loading classifier", err)
	}

	tc = sms.NewClient()
	mc = dt.NewMailClient()

	appVocab = dt.NewAtomicMap()
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")

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
	if len(cmd) >= 5 && strings.ToLower(cmd)[0:5] == "train" {
		if err := train(bayes, cmd[6:]); err != nil {
			return nil, err
		}
		return nil, nil
	}
	uid, fid, fidT := validateParams(c)
	u, err := getUser(uid, fid, flexIDType(fidT))
	if err != nil {
		return nil, err
	}
	msg := dt.NewMsg(db, bayes, u, cmd)
	if err = msg.Update(db); err != nil {
		return nil, err
	}
	// TODO trigger training if needed (see buildInput)
	return msg, nil
}

func processKnowledge(msg *dt.Msg, ret *dt.RespMsg, followup bool) (*dt.RespMsg,
	bool, error) {
	var edges []*edge
	var err error
	if len(ret.Sentence) == 0 {
		edges, err = searchEdgesForTerm(msg.Sentence)
		if err != nil {
			return nil, false, err
		}
		for _, e := range edges {
			msg, ret, err = processAgain(msg, e, followup)
			if err != nil {
				return nil, false, err
			}
			if len(ret.Sentence) > 0 {
				e.IncrementConfidence(db)
				break
			}
			e.DecrementConfidence(db)
		}
	}
	var nodes []*node
	if len(ret.Sentence) == 0 {
		nodes, err = searchNodes(msg.Sentence, int64(len(edges)))
		if err != nil {
			return nil, false, err
		}
		for _, n := range nodes {
			if len(n.Rel()) == 0 {
				break
			}
			msg, ret, err = processAgain(msg, n, followup)
			if err != nil {
				return nil, false, err
			}
			if len(ret.Sentence) > 0 {
				n.IncrementConfidence(db)
				break
			}
			n.DecrementConfidence(db)
		}
	}
	log.Debugln("nodes found", nodes)
	log.Debugln("ret.Sentence", ret.Sentence)
	if len(ret.Sentence) == 0 && len(nodes) == 0 {
		nodes, err := newNodes(db, appVocab, msg)
		if err != nil {
			return nil, false, err
		}
		if len(nodes) > 0 {
			log.Debugln("created nodes, still need to save")
			ret.Sentence = nodes[0].Text()
			return ret, false, nil
		}
	}
	var changed bool
	if len(nodes) > 0 {
		msg, ret, err = processAgain(msg, nodes[0], followup)
		if err != nil {
			return nil, false, err
		}
		changed = true
	}
	return ret, changed, nil
}

func processAgain(msg *dt.Msg, g graphObj, followup bool) (*dt.Msg, *dt.RespMsg,
	error) {
	var err error
	msg.Sentence, err = replaceSentence(db, msg, g)
	if err != nil {
		return msg, nil, err
	}
	si, _, _, err := nlp.Classify(bayes, msg.Sentence)
	if err != nil {
		log.Errorln("classifying sentence", err)
	}
	pkg, route, _, err := getPkg(msg)
	if err != nil && err != ErrMissingPackage {
		log.WithField("fn", "getPkg").Error(err)
		return msg, nil, err
	}
	msg = dt.NewMsg(db, bayes, msg.User, msg.Sentence)
	msg.StructuredInput = si
	msg.Route = route
	ret, err := callPkg(pkg, msg, followup)
	if err != nil {
		return msg, nil, err
	}
	return msg, ret, nil
}

func processText(c *echo.Context) (string, error) {
	msg, err := preprocess(c)
	if err != nil || msg == nil /* trained */ {
		log.WithField("fn", "preprocessForMessage").Error(err)
		return "", err
	}

	log.Debugln("processed input into message...")
	log.Debugln("commands:", msg.StructuredInput.Commands)
	log.Debugln(" objects:", msg.StructuredInput.Objects)
	log.Debugln("  actors:", msg.StructuredInput.Actors)
	log.Debugln("   times:", msg.StructuredInput.Times)
	log.Debugln("  places:", msg.StructuredInput.Places)

	pkg, route, followup, err := getPkg(msg)
	if err != nil {
		log.WithField("fn", "getPkg").Error(err)
		return "", err
	}
	msg.Route = route
	msg.Package = pkg.P.Config.Name
	if err = msg.Save(db); err != nil {
		return "", err
	}
	ret, err := callPkg(pkg, msg, followup)
	if err != nil {
		return "", err
	}
	var m *dt.Msg
	if len(ret.Sentence) == 0 {
		m = &dt.Msg{}
		m.Sentence = language.Confused()
		m.AvaSent = true
		m.User = msg.User
		if err = m.Save(db); err != nil {
			return "", err
		}
	}
	if m == nil {
		m, err = dt.GetMsg(db, ret.MsgID)
		if err != nil {
			return "", err
		}
	}
	if pkg != nil {
		m.Package = pkg.P.Config.Name
	}
	if m.ID > 0 {
		if err = m.Update(db); err != nil {
			return "", err
		}
	} else {
		if err = m.Save(db); err != nil {
			return "", err
		}
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
	return m.Sentence, nil
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
			log.WithField("fn", "validateParams").Fatalln(err)
		}
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
