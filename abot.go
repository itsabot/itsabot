package main

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/codegangsta/cli"
	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/pkg"
	"github.com/itsabot/abot/shared/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	_ "github.com/lib/pq"
)

var db *sqlx.DB
var ner core.Classifier
var ws = websocket.NewAtomicWebSocketSet()
var offensive map[string]struct{}
var (
	errInvalidUserPass = errors.New("Invalid username/password combination")
)

func main() {
	rand.Seed(time.Now().UnixNano())
	log.DebugOn(true)
	app := cli.NewApp()
	app.Name = "abot"
	app.Usage = "digital assistant framework"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "server, s",
			Usage: "run server",
		},
		cli.BoolFlag{
			Name:  "install, i",
			Usage: "install packages in packages.json",
		},
		cli.BoolFlag{
			Name:  "console, c",
			Usage: "communicate with a running abot server",
		},
	}
	app.Action = func(c *cli.Context) {
		showHelp := true
		if c.Bool("install") {
			log.Info("TODO: install packages")
			showHelp = false
		}
		if c.Bool("console") {
			log.Info("TODO: run console")
			showHelp = false
		}
		if c.Bool("server") {
			var err error
			db, err = pkg.ConnectDB()
			if err != nil {
				log.Fatal("could not connect to database", err)
			}
			if err = startServer(); err != nil {
				log.Fatal("could not start server", err)
			}
			showHelp = false
		}
		if showHelp {
			cli.ShowAppHelp(c)
		}
	}
	app.Run(os.Args)
}

// startServer initializes any clients that are needed and boots packages
func startServer() error {
	if err := checkRequiredEnvVars(); err != nil {
		return err
	}
	addr, err := core.BootRPCServer()
	if err != nil {
		return err
	}
	go func() {
		if err := core.BootDependencies(addr); err != nil {
			log.Debug("could not boot dependency", err)
		}
	}()
	ner, err = core.BuildClassifier()
	if err != nil {
		log.Debug("could not build classifier", err)
	}
	offensive, err = core.BuildOffensiveMap()
	if err != nil {
		log.Debug("could not build offensive map", err)
	}
	e := echo.New()
	initRoutes(e)
	log.Info("booted ava http server")
	e.Run(":" + os.Getenv("ABOT_PORT"))
	return nil
}

func checkRequiredEnvVars() error {
	port := os.Getenv("ABOT_PORT")
	_, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("ABOT_PORT is not set to an integer")
	}
	base := os.Getenv("ABOT_URL")
	l := len(base)
	if l == 0 {
		return errors.New("ABOT_URL not set")
	}
	if l < 4 || base[0:4] != "http" {
		return errors.New("ABOT_URL invalid. Must include http/https")
	}
	// TODO Check for ABOT_DATABASE_URL if AVA_ENV==production
	return nil
}
