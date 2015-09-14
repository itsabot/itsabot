package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/codegangsta/cli"
	fnlp "github.com/egtann/freeling/nlp"
	"github.com/jbrukh/bayesian"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
)

var db *sqlx.DB
var nlp *fnlp.NLPEngine

var ErrInvalidCommand = errors.New("invalid command")

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

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
	db = connectDB()
	// Load packages
	/*
		bc, err := loadConfig("packages.conf")
		if err != nil {
			log.Fatalln("could not load package", err)
		}
	*/
	opts := fnlp.NewNLPOptions(path.Join(".", "data"), "en", func() { log.Println("hit") })
	nlp = fnlp.NewNLPEngine(opts)
	e := echo.New()
	initRoutes(e)
	e.Run(":" + port)
}

func loadConfig(p string) (map[string]bayesian.Class, error) {
	bc := map[string]bayesian.Class{}
	content, err := ioutil.ReadFile(p)
	if err != nil {
		return bc, err
	}
	pkgs := strings.Split(string(content), "\n")
	for _, pkg := range pkgs {
		bc[pkg] = bayesian.Class(pkg)
	}
	return bc, nil
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
	cmd := c.Form("cmd")
	if len(cmd) == 0 {
		return ErrInvalidCommand
	}
	// Update state machine
	// Save last command (save structured input)

	si := buildStructuredInput(cmd)
	log.Println("structured input", si)

	// Send to packages
	ret := ""
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
	err := c.HTML(http.StatusOK, ret)
	if err != nil {
		return err
	}
	return nil
}
