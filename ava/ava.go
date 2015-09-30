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
	"github.com/codegangsta/cli"
	"github.com/jbrukh/bayesian"
	"github.com/jmoiron/sqlx"
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
			log.Info("TODO: Install packages")
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
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Error("loading classifier: ", err)
	}
	go bootRPCServer(port)
	bootDependencies()
	log.Debug("booting local server")
	e := echo.New()
	initRoutes(e)
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
	l, err := net.Listen("tcp", ":"+strconv.Itoa(p+1))
	if err != nil {
		log.Error("rpc listen: ", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Error("rpc accept: ", err)
		}
		go rpc.ServeConn(conn)
	}
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
}

func handlerMain(c *echo.Context) error {
	var ret string
	var err error
	si := &datatypes.StructuredInput{}
	cmd := c.Form("cmd")
	if len(cmd) == 0 {
		return ErrInvalidCommand
	}
	if strings.ToLower(cmd)[0:5] == "train" {
		if err := train(bayes, cmd[7:]); err != nil {
			return err
		}
		goto Response
	}
	si, err = classify(bayes, cmd)
	if err != nil {
		log.Error("error classifying sentence", err)
	}
	ret, err = callPkg(c.Form("id"), si)
	if err != nil {
		return err
	}
	// Update state machine
	// Save last command (save structured input)
Response:
	err = c.HTML(http.StatusOK, ret)
	if err != nil {
		return err
	}
	return nil
}
