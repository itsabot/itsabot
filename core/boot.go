package core

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/itsabot/abot/shared/log"
	"github.com/itsabot/abot/shared/plugin"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
)

var db *sqlx.DB
var ner Classifier
var offensive map[string]struct{}

func DB() *sqlx.DB {
	return db
}

func NER() Classifier {
	return ner
}

func Offensive() map[string]struct{} {
	return offensive
}

// NewServer connects to the database and boots all plugins before returning a
// server connection, database connection, and map of offensive words.
func NewServer() (e *echo.Echo, abot *Abot, rpcAddr string, err error) {
	if len(os.Getenv("ABOT_SECRET")) < 32 && os.Getenv("ABOT_ENV") == "production" {
		return nil, abot, "", errors.New("must set ABOT_SECRET env var in production to >= 32 characters")
	}
	db, err = plugin.ConnectDB()
	if err != nil {
		return nil, abot, "", fmt.Errorf("could not connect to database: %s", err.Error())
	}
	if err = checkRequiredEnvVars(); err != nil {
		return nil, abot, "", err
	}
	if os.Getenv("ABOT_ENV") != "test" {
		if err = CompileAssets(); err != nil {
			return nil, abot, "", err
		}
	}
	abot, rpcAddr, err = BootRPCServer()
	if err != nil {
		return nil, abot, "", err
	}
	go func() {
		if err = BootDependencies(rpcAddr); err != nil {
			log.Debug("could not boot dependency", err)
		}
	}()
	ner, err = buildClassifier()
	if err != nil {
		log.Debug("could not build classifier", err)
	}
	offensive, err = buildOffensiveMap()
	if err != nil {
		log.Debug("could not build offensive map", err)
	}
	e = echo.New()
	p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "itsabot",
		"abot", "assets", "html", "layout.html")
	if err = loadHTMLTemplate(p); err != nil {
		return nil, abot, "", err
	}
	initRoutes(e)
	return e, abot, rpcAddr, nil
}

// CompileAssets compresses and merges assets from Abot core and all plugins on
// boot. In development, this step is repeated on each server HTTP request prior
// to serving any assets.
func CompileAssets() error {
	outC, err := exec.
		Command("/bin/sh", "-c", "cmd/compileassets.sh").
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
