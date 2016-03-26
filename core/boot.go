package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/itsabot/abot/core/log"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq" // Postgres driver
)

var db *sqlx.DB
var ner Classifier
var offensive map[string]struct{}

// DB returns a connection to the database.
func DB() *sqlx.DB {
	return db
}

// NER returns the classifiers used for named entity recognition.
func NER() Classifier {
	return ner
}

// Offensive returns a map of offensive words that Abot should ignore.
func Offensive() map[string]struct{} {
	return offensive
}

// NewServer connects to the database and boots all plugins before returning a
// server connection, database connection, and map of offensive words.
func NewServer() (r *httprouter.Router, err error) {
	if len(os.Getenv("ABOT_SECRET")) < 32 && os.Getenv("ABOT_ENV") == "production" {
		return nil, errors.New("must set ABOT_SECRET env var in production to >= 33 characters")
	}
	if db == nil {
		db, err = ConnectDB()
		if err != nil {
			return nil, fmt.Errorf("could not connect to database: %s", err.Error())
		}
	}
	if err = checkRequiredEnvVars(); err != nil {
		return nil, err
	}

	// Get ImportPath from plugins.json
	conf, err := LoadConf()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(os.Getenv("GOPATH"), "src", conf.ImportPath)

	if err = os.Setenv("ABOT_PATH", p); err != nil {
		return nil, err
	}
	if os.Getenv("ABOT_ENV") != "test" {
		if err = CompileAssets(); err != nil {
			return nil, err
		}
	}
	ner, err = buildClassifier()
	if err != nil {
		log.Debug("could not build classifier", err)
	}
	offensive, err = buildOffensiveMap()
	if err != nil {
		log.Debug("could not build offensive map", err)
	}
	p = filepath.Join(p, "assets", "html", "layout.html")
	if err = loadHTMLTemplate(p); err != nil {
		return nil, err
	}
	return newRouter(), nil
}

// CompileAssets compresses and merges assets from Abot core and all plugins on
// boot. In development, this step is repeated on each server HTTP request prior
// to serving any assets.
func CompileAssets() error {
	p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "itsabot",
		"abot", "cmd", "compileassets.sh")
	outC, err := exec.
		Command("/bin/sh", "-c", p).
		CombinedOutput()
	if err != nil {
		log.Debug(string(outC))
		return err
	}
	return nil
}

func loadHTMLTemplate(p string) error {
	var err error
	tmplLayout, err = template.ParseFiles(p)
	return err
}

func checkRequiredEnvVars() error {
	port := os.Getenv("PORT")
	_, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("PORT is not set to an integer")
	}
	base := os.Getenv("ABOT_URL")
	l := len(base)
	if l == 0 {
		return errors.New("ABOT_URL not set")
	}
	if l < 4 || base[0:4] != "http" {
		return errors.New("ABOT_URL invalid. Must include http/https")
	}
	// TODO Check for ABOT_DATABASE_URL if ABOT_ENV==production
	return nil
}

// ConnectDB opens a connection to the database.
func ConnectDB() (*sqlx.DB, error) {
	var d *sqlx.DB
	var err error
	if os.Getenv("ABOT_ENV") == "production" {
		d, err = sqlx.Connect("postgres", os.Getenv("ABOT_DATABASE_URL"))
	} else if os.Getenv("ABOT_ENV") == "test" {
		d, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot_test sslmode=disable")
	} else {
		d, err = sqlx.Connect("postgres",
			"user=postgres dbname=abot sslmode=disable")
	}
	return d, err
}

// LoadConf plugins.json into a usable struct.
func LoadConf() (*PluginJSON, error) {
	contents, err := ioutil.ReadFile("plugins.json")
	if err != nil {
		if err.Error() != "open plugins.json: no such file or directory" {
			return nil, err
		}
		contents, err = ioutil.ReadFile(filepath.Join("..", "plugins.json"))
		if err != nil {
			return nil, err
		}
	}
	plugins := &PluginJSON{}
	if err = json.Unmarshal(contents, plugins); err != nil {
		return nil, err
	}
	return plugins, nil
}
