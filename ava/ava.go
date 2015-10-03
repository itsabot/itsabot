package main

import (
	"errors"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
	"github.com/codegangsta/cli"
	"github.com/jbrukh/bayesian"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
)

var db *sqlx.DB
var bayes *bayesian.Classifier
var ErrInvalidCommand = errors.New("invalid command")
var ErrMissingPackage = errors.New("missing package")

func main() {
	rand.Seed(time.Now().UnixNano())
	if os.Getenv("AVA_ENV") == "production" {
		log.SetLevel(log.WarnLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}
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
			Value: "4000",
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
			startServer(c.String("port"))
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
	if err = godotenv.Load(); err != nil {
		log.Error("loading environment: ", err)
	}
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Error("loading classifier: ", err)
	}
	log.Debug("booting local server")
	bootRPCServer(port)
	bootTwilio()
	bootDependencies()
	e := echo.New()
	initRoutes(e)
	log.Info("booted ava")
	e.Run(":" + port)
}

func bootRPCServer(port string) {
	ava := new(Ava)
	if err := rpc.Register(ava); err != nil {
		log.Error("register ava in rpc", err)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		log.Error("convert port to int", err)
	}
	pt := strconv.Itoa(p + 1)
	l, err := net.Listen("tcp", ":"+pt)
	log.WithField("port", pt).Debug("booting rpc server")
	if err != nil {
		log.Error("rpc listen: ", err)
	}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Error("rpc accept: ", err)
			}
			go rpc.ServeConn(conn)
		}
	}()
}

func connectDB() *sqlx.DB {
	log.Debug("connecting to db")
	db, err := sqlx.Connect("postgres",
		"user=egtann dbname=ava sslmode=disable")
	if err != nil {
		log.Error("could not connect to db ", err.Error())
	}
	return db
}

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger())
	e.Use(mw.Gzip())
	e.Use(mw.Recover())
	e.SetDebug(true)
	e.Post("/", handlerMain)
	e.Post("/twilio", handlerTwilio)
}

// TODO
func handlerTwilio(c *echo.Context) error {
	log.Error("twilio endpoint not implemented")
	return errors.New("not implemented")
}

func handlerMain(c *echo.Context) error {
	cmd := c.Form("cmd")
	if len(cmd) == 0 {
		return ErrInvalidCommand
	}
	var ret, pname, route string
	var err error
	var uid, fidT int
	var ctxAdded bool
	var pw *pkg.PkgWrapper
	si := &datatypes.StructuredInput{}
	if len(cmd) >= 5 && strings.ToLower(cmd)[0:5] == "train" {
		if err := train(bayes, cmd[6:]); err != nil {
			return err
		}
		goto Response
	}
	si, err = classify(bayes, cmd)
	if err != nil {
		log.Error("classifying sentence ", err)
	}
	uid, err = strconv.Atoi(c.Form("uid"))
	if err.Error() == `strconv.ParseInt: parsing "": invalid syntax` {
		uid = 0
	} else if err != nil {
		return err
	}
	fidT, err = strconv.Atoi(c.Form("flexidtype"))
	if err != nil && err.Error() == `strconv.ParseInt: parsing "": invalid syntax` {
		fidT = 0
	} else if err != nil {
		return err
	}
	si, ctxAdded, err = addContext(si, uid, c.Form("flexid"), fidT)
	if err != nil {
		log.Error("adding context ", err)
	}
	pw, route, err = getPkg(si)
	if err != nil && err.Error() != "missing package" {
		return err
	}
	if pw != nil {
		ret, err = callPkg(pw, si, ctxAdded)
		if err != nil && err.Error() != "missing package" {
			return err
		}
	}
	if len(ret) == 0 {
		ret = language.Confused()
	}
	if pw != nil {
		pname = pw.P.Config.Name
	}
	if err := saveStructuredInput(si, ret, pname, route); err != nil {
		return err
	}
Response:
	err = c.HTML(http.StatusOK, ret)
	if err != nil {
		return err
	}
	return nil
}
