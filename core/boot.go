package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/interface/email"
	"github.com/itsabot/abot/shared/interface/sms"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq" // Postgres driver
)

var db *sqlx.DB
var ner Classifier
var offensive map[string]struct{}
var smsConn *sms.Conn
var emailConn *email.Conn

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

	conf, err := LoadConf()
	if err != nil && os.Getenv("ABOT_ENV") != "test" {
		log.Info("failed loading conf", err)
		return nil, err
	}
	var p string
	if err == nil {
		p = filepath.Join(os.Getenv("GOPATH"), "src", conf.ImportPath)
		if err = os.Setenv("ABOT_PATH", p); err != nil {
			return nil, err
		}
	}

	/*
		if os.Getenv("ABOT_ENV") != "test" {
			if err = CompileAssets(); err != nil {
				return nil, err
			}
		}
	*/
	ner, err = buildClassifier()
	if err != nil {
		log.Debug("could not build classifier", err)
	}
	offensive, err = buildOffensiveMap()
	if err != nil {
		log.Debug("could not build offensive map", err)
	}
	p2 := filepath.Join("assets", "html", "layout.html")
	if err = loadHTMLTemplate(p, p2); err != nil {
		log.Info("failed loading HTML template", err)
		return nil, err
	}

	// Initialize a router with routes
	r = newRouter()

	// Open a connection to an SMS service
	if len(sms.Drivers()) > 0 {
		drv := sms.Drivers()[0]
		smsConn, err = sms.Open(drv, r)
		if err != nil {
			log.Info("failed to open sms driver connection", drv,
				err)
		}
	} else {
		log.Debug("no sms drivers imported")
	}

	// Open a connection to an email service
	if len(email.Drivers()) > 0 {
		drv := email.Drivers()[0]
		emailConn, err = email.Open(drv, r)
		if err != nil {
			log.Info("failed to open email driver connection", drv,
				err)
		}
	} else {
		log.Debug("no email drivers imported")
	}

	// Listen for events that need to be sent.
	evtChan := make(chan *dt.ScheduledEvent)
	go func(chan *dt.ScheduledEvent) {
		q := `UPDATE scheduledevents SET sent=TRUE WHERE id=$1`
		select {
		case evt := <-evtChan:
			log.Debug("received event")
			// Send event. On error, event will be retried next
			// minute.
			if err := evt.Send(smsConn); err != nil {
				log.Info("failed to send scheduled event", err)
				return
			}
			// Update event as sent
			if _, err := db.Exec(q); err != nil {
				log.Info("failed to update scheduled event as sent",
					err)
				return
			}
		}
	}(evtChan)

	// Check every minute if there are any scheduled events that need to be
	// sent.
	go func(evtChan chan *dt.ScheduledEvent) {
		q := `SELECT id, content, flexid, flexidtype
		      FROM scheduledevents
		      WHERE sent=false AND sendat<=$1`
		t := time.NewTicker(time.Minute)
		select {
		case now := <-t.C:
			evts := []*dt.ScheduledEvent{}
			if err := db.Select(&evts, q, now); err != nil {
				log.Info("failed to queue scheduled event", err)
				return
			}
			for _, evt := range evts {
				// Queue the event for sending
				evtChan <- evt
			}
		}
	}(evtChan)

	return r, nil
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

func loadHTMLTemplate(p, p2 string) error {
	var err error
	tmplLayout, err = template.ParseFiles(filepath.Join(p, p2))
	if tmplLayout == nil {
		tmplLayout, err = template.ParseFiles(p2)
	}
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
	dbConnStr := os.Getenv("ABOT_DATABASE_URL")
	if dbConnStr == "" {
		dbConnStr = "host=127.0.0.1 user=postgres"
	}
	if len(dbConnStr) <= 11 || dbConnStr[:11] != "postgres://" {
		dbConnStr += " sslmode=disable dbname=abot"
		if strings.ToLower(os.Getenv("ABOT_ENV")) == "test" {
			dbConnStr += "_test"
		}
	}
	return sqlx.Connect("postgres", dbConnStr)
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

// LoadEnvVars from abot.env into memory
func LoadEnvVars() error {
	if len(os.Getenv("ITSABOT_URL")) == 0 {
		log.Debug("ITSABOT_URL not set, using https://www.itsabot.org")
		err := os.Setenv("ITSABOT_URL", "https://www.itsabot.org")
		if err != nil {
			return err
		}
	}
	p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "itsabot",
		"abot", "abot.env")
	fi, err := os.Open(p)
	if os.IsNotExist(err) {
		// Assume the user has loaded their env variables into their
		// path
		return nil
	}
	if err != nil {
		return err
	}
	defer func() {
		if err = fi.Close(); err != nil {
			log.Info("failed to close file")
		}
	}()
	scn := bufio.NewScanner(fi)
	for scn.Scan() {
		line := scn.Text()
		fields := strings.SplitN(line, "=", 2)
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(os.Getenv(key))
		if val == "" {
			val = strings.TrimSpace(fields[1])
			if err = os.Setenv(key, val); err != nil {
				return err
			}
		}
	}
	if err = scn.Err(); err != nil {
		return err
	}
	return nil
}
