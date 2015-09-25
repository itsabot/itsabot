package main

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

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

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
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
			Value: "4000",
			Usage: "set port for server",
		},
		cli.BoolFlag{
			Name:  "install, i",
			Usage: "install packages in package.conf",
		},
	}
	app.Action = func(c *cli.Context) {
		showHelp := true
		if c.Bool("install") {
			log.Println("TODO: Install packages")
			showHelp = false
		}
		if c.Bool("server") {
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
	db = connectDB()
	// Load packages
	/*
		bc, err := loadConfig("packages.conf")
		if err != nil {
			log.Fatalln("could not load package", err)
		}
	*/
	bayes, err = loadClassifier(bayes)
	if err != nil {
		log.Fatalln("error loading classifier", err)
	}
	/*
		si, err := classify(bayes, "train _C(Order) _O(an Uber).")
		if err != nil {
			log.Fatalln("error classifying sentence", err)
		}
		log.Println(si)
	*/

	e := echo.New()
	initRoutes(e)
	e.Run(":" + port)
}

// route will determine what kind of request it is based on text.
// Content can belong to multiple classes. Route returns []string,
// which is used by os.Exec to run the commands.
func route(content string) []string {
	var pkgs []string

	return pkgs
}

func connectDB() *sqlx.DB {
	db, err := sqlx.Connect("postgres",
		"user=egtann dbname=ava sslmode=disable")
	if err != nil {
		log.Fatalln(err)
	}
	return db
	// Run schema while testing
}

func initRoutes(e *echo.Echo) {
	e.Use(mw.Logger())
	e.Use(mw.Gzip())
	e.Use(mw.Recover())
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
		log.Fatalln("error classifying sentence", err)
	}
	ret = si.String()
	// Update state machine
	// Save last command (save structured input)
	// Send to packages
	/*
		for _, pkg := range pkgs {
			path := path.Join("packages", pkg)
			out, err := exec.Command(path, cmd).CombinedOutput()
			if err != nil {
				log.Println("unable to run package", err)
				return err
			}
			ret += string(out) + "\n\n"
		}
	*/

Response:
	err = c.HTML(http.StatusOK, ret)
	if err != nil {
		return err
	}
	return nil
}
