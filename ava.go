package main

import (
	"database/sql"
	"errors"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
	"github.com/avabot/ava/shared/knowledge"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
	"github.com/avabot/ava/shared/sms"
)

// TODO variable routes. e.g. "Help me get drunk" could route to purchase
// (alcohol) or bars nearby. Ava should ask the user which route to send them
// to on packages with overlapping routes.

var db *sqlx.DB
var tc *twilio.Client
var mc *dt.MailClient
var bayes *bayesian.Classifier
var phoneRegex *regexp.Regexp
var ErrInvalidCommand = errors.New("invalid command")
var ErrMissingPackage = errors.New("missing package")
var ErrInvalidUserPass = errors.New("Invalid username/password combination")

type Ctx struct {
	Msg           *dt.Msg
	Input         *dt.Input
	User          *dt.User
	NeedsTraining bool
}

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
		log.Errorln("checking env vars", err)
	}
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Errorln("loading classifier", err)
	}
	bootRPCServer(port)
	tc = sms.NewClient()
	mc = dt.NewMailClient()
	appVocab = atomicMap{
		words: map[string]bool{},
		mutex: &sync.Mutex{},
	}
	bootDependencies()
	stripe.Key = os.Getenv("STRIPE_ACCESS_TOKEN")
	e := echo.New()
	initRoutes(e)
	log.Infoln("booted ava")
	e.Run(":" + port)
}

func bootRPCServer(port string) {
	ava := new(Ava)
	if err := rpc.Register(ava); err != nil {
		log.Errorln("register ava in rpc", err)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		log.Errorln("convert port to int", err)
	}
	pt := strconv.Itoa(p + 1)
	l, err := net.Listen("tcp", ":"+pt)
	log.WithFields(log.Fields{
		"port": pt,
	}).Debugln("booting rpc server")
	if err != nil {
		log.Errorln("rpc listen", err)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Errorln("rpc accept", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
}

func connectDB() *sqlx.DB {
	log.Debugln("connecting to db")
	var d *sqlx.DB
	var err error
	if os.Getenv("AVA_ENV") == "production" {
		d, err = sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	} else {
		d, err = sqlx.Connect("postgres",
			"user=egtann dbname=ava sslmode=disable")
	}
	if err != nil {
		log.Errorln("connecting to db", err)
	}
	log.Infoln("connected to db")
	return d
}

func fillInWithKnowledge(c *echo.Context, p *pkg.Pkg, m *dt.Msg) (string,
	error) {
	log.Debugln("filling in with knowledge")
	/*
		if m.LastInput.KnowledgeFilled {
			log.Debugln("last input was knowledgefilled")
			return "", nil
		}
	*/
	// TODO determine why this reports changed as false when the
	// knowledgequery DOES exist
	sentence, changed, err := knowledge.FillIn(db,
		m.Input.StructuredInput.Objects.StringSlice(), m.Input.Sentence,
		m.User)
	if err != nil {
		return sentence, err
	}
	if changed {
		log.Debugln("before", m.Input.Sentence)
		log.Debugln("after", sentence)
		m.Input.KnowledgeFilled = true
		if err = m.Input.Save(db); err != nil {
			return sentence, err
		}
		return sentence, nil
	}
	if err := m.GetLastInput(db); err != nil {
		return "", err
	}
	err = knowledge.SolveLastQuery(db, m)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	sentence = m.Input.Sentence
	if err == sql.ErrNoRows {
		qs, err := knowledge.NewQueriesForPkg(db, p, m)
		if err != nil {
			return "", err
		}
		return qs[0].Text(), nil
	} else {
		log.Debugln("filling in sentence")
		var changed bool
		c.Set("cmd", m.LastInput.Sentence)
		si, _, _, err := classify(bayes, c.Get("cmd").(string))
		if err != nil {
			log.Errorln("classifying lastinput", err)
			return sentence, err
		}
		sentence, changed, err = knowledge.FillIn(db,
			si.Objects.StringSlice(), m.LastInput.Sentence, m.User)
		if err != nil {
			return sentence, err
		}
		log.Debugln("before", m.Input.Sentence)
		log.Debugln("after", sentence)
		if changed {
			m.Input.KnowledgeFilled = true
			if err = m.LastInput.Save(db); err != nil {
				return sentence, err
			}
			return sentence, nil
		} else {
			log.Warnln("not changed")
		}
	}
	return sentence, nil
}

func preprocess(c *echo.Context) (*Ctx, error) {
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
	si, annotated, needsTraining, err := classify(bayes, cmd)
	if err != nil {
		log.Errorln("classifying sentence", err)
	}
	in := &dt.Input{
		Sentence:          cmd,
		StructuredInput:   si,
		FlexID:            fid,
		FlexIDType:        fidT,
		UserID:            uid,
		SentenceAnnotated: annotated,
	}
	u, err := getUser(in)
	if err == dt.ErrMissingUser {
		log.Infoln("missing user", err)
	} else if err != nil {
		log.WithField("fn", "getUser").Errorln(err)
		return nil, err
	}
	in.UserID = u.ID
	in.StructuredInput = si
	ctx := &Ctx{
		Input:         in,
		User:          u,
		Msg:           dt.NewMessage(db, u, in),
		NeedsTraining: needsTraining,
	}
	return ctx, nil
}

// TODO mark a knowledgequery with a liveat timestamp if the package returns a
// successful response. Then when searching knowledgequeries, order by the most
// recent liveat. Prevents an old (and since outdated) knowledgequery from
// blocking new, successful ones.
func processText(c *echo.Context) (string, error) {
	ctx, err := preprocess(c)
	if err != nil || ctx == nil /* trained */ {
		log.WithField("fn", "preprocessForMessage").Error(err)
		return "", err
	}
	pkg, route, followup, err := getPkg(ctx.Msg)
	if err != nil {
		log.WithField("fn", "getPkg").Error(err)
		return "", err
	}
	var filledInWithKnowledge bool
	var lastQRID uint64
	if !followup {
		log.Debugln("conversation change. deleting unused knowledgequeries")
		if err := knowledge.DeleteQueries(db, ctx.User); err != nil {
			return "", err
		}
	} else {
		lastQRID, err = knowledge.LastQueryResponseID(db, ctx.Msg)
		if err != nil {
			return "", err
		}
	}
	if lastQRID > 0 && ctx.Msg.LastResponse.ID == lastQRID {
		filledInWithKnowledge = true
		sent, err := fillInWithKnowledge(c, pkg.P, ctx.Msg)
		if err != nil {
			log.Errorln("fillInWithKnowledge", err)
			return "", err
		}
		log.Debugln("changed sentence", sent)
		c.Set("cmd", sent)
		ctx, err = preprocess(c)
		if err != nil {
			log.Errorln("preprocessForMessage", err)
			return "", err
		}
	}
	ctx.Msg.Route = route
	// callPkg nils out lastResponse for rpc gob transfer, so we save a
	// reference to it here
	lastResponse := ctx.Msg.LastResponse
	ret, err := callPkg(pkg, ctx.Msg, followup)
	if err != nil && err != ErrMissingPackage {
		log.WithField("fn", "callPkg").Errorln(err)
		return "", err
	}
	ctx.Msg.LastResponse = lastResponse
	var confused bool
	if len(ret.Sentence) == 0 {
		log.Debugln("pkg response empty")
		// fill in learned knowledge of language and try again
		if !filledInWithKnowledge {
			// TODO pass back changed bool
			sent, err := fillInWithKnowledge(c, pkg.P, ctx.Msg)
			if err != nil {
				log.Errorln("fillInWithKnowledge", err)
			}
			// TODO use changed bool
			if sent != ctx.Msg.Input.Sentence {
				c.Set("cmd", sent)
				return processText(c)
			}
		}
		if len(ret.Sentence) == 0 {
			ret.Sentence = language.Confused()
		}
	}
	id, err := saveStructuredInput(ctx.Msg, ret.ResponseID,
		pkg.P.Config.Name, route)
	if err != nil {
		return ret.Sentence, err
	}
	if confused && !followup {
		log.WithField("inputID", id).Infoln("confused")
		// TODO allow for fuzzy matching. For example
		//
		// User> "cab sau"
		// Ava>  Did you mean Cabernet Sauvignon?
		if len(ctx.Input.StructuredInput.Commands) > 0 ||
			len(ctx.Input.StructuredInput.Objects) > 0 {
			qs, err := knowledge.NewQueriesForPkg(db, pkg.P,
				ctx.Msg)
			if err != nil {
				return ret.Sentence, err
			}
			return qs[0].Text(), nil
		}
	}
	ctx.Input.ID = id
	if ctx.NeedsTraining {
		log.WithField("inputID", id).Infoln("needed training")
		if err = supervisedTrain(ctx.Input); err != nil {
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
