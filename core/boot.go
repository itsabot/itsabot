package core

import (
	"errors"
	"os"
	"os/exec"
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
func NewServer() (*echo.Echo, error) {
	if len(os.Getenv("ABOT_SECRET")) < 32 && os.Getenv("ABOT_ENV") == "production" {
		log.Fatal("must set ABOT_SECRET env var in production to >= 32 characters")
	}
	var err error
	db, err = plugin.ConnectDB()
	if err != nil {
		log.Fatal("could not connect to database", err)
	}
	if err = checkRequiredEnvVars(); err != nil {
		return nil, err
	}
	if os.Getenv("ABOT_ENV") != "test" {
		if err = CompileAssets(); err != nil {
			return nil, err
		}
	}
	addr, err := BootRPCServer()
	if err != nil {
		return nil, err
	}
	go func() {
		if err = BootDependencies(addr); err != nil {
			log.Debug("could not boot dependency", err)
		}
	}()
	if os.Getenv("ABOT_ENV") != "test" {
		ner, err = BuildClassifier()
		if err != nil {
			log.Debug("could not build classifier", err)
		}
		offensive, err = BuildOffensiveMap()
		if err != nil {
			log.Debug("could not build offensive map", err)
		}
	}
	e := echo.New()
	return e, nil
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
		log.Fatal(err)
	}
	return nil
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
