package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/jbrukh/bayesian"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	_ "github.com/lib/pq"
)

// #cgo CFLAGS: -I .
// #cgo LDFLAGS: -L . -lmitie
// #include <mitie.h>
import "C"

var classifier *bayesian.Classifier
var db *sqlx.DB

var ErrInvalidCommand = errors.New("invalid command")

var dict wMap

// NOTE: Arbitrary. Will be adjusted with learning data.
const ClassifierThreshold = 0.7

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
	bc, err := loadConfig("packages.conf")
	if err != nil {
		log.Fatalln("could not load package", err)
	}
	// TODO: ensure all installed

	//loadModel("data/mitie_ner.dat")
	dict, err = loadDictionary()
	if err != nil {
		log.Fatalln("could not load dictionaries", err)
	}

	classifier, err = trainedClassifier(bc)
	if err != nil {
		log.Fatalln("could not retrieve trained classifier", err)
	}

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
	cn := strings.Fields(content)
	probs, _, _ := classifier.ProbScores(cn)
	var pkgs []string
	for i, prob := range probs {
		log.Println("Class probability:", prob, string(classifier.Classes[i]))
		if prob > ClassifierThreshold {
			pkgs = append(pkgs, string(classifier.Classes[i]))
		}
	}
	return pkgs
}

func connectDB() *sqlx.DB {
	db, err := sqlx.Connect("postgres", "user=egtann dbname=ava sslmode=disable")
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

	// Route with Bayes
	pkgs := route(cmd)
	log.Println("routing to", pkgs)

	// Update state machine
	// Save last command. Save nouns/context.
	// NOTE: This has a JDK 8 dependency, which I'll aim to remove in subsequent versions.
	// Grab objects of prepositions (times), people, organizations, locations.
	si := buildStructuredInput(cmd)
	log.Println("structured input", si)

	// Send to packages
	ret := ""
	for _, pkg := range pkgs {
		path := path.Join("packages", pkg)
		out, err := exec.Command(path, cmd).CombinedOutput()
		if err != nil {
			log.Println("unable to run package", err)
			return err
		}
		ret += string(out) + "\n\n"
	}

	err := c.HTML(http.StatusOK, ret)
	if err != nil {
		return err
	}

	return nil
}

func loadModel(p string) {
	ner := C.mitie_load_named_entity_extractor(C.CString(p))
	if ner == nil {
		log.Println("unable to load model file")
		return
	}
}
