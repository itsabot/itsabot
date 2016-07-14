package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
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
	"github.com/jbrukh/bayesian"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq" // Postgres driver
)

var db *sqlx.DB
var ner classifier
var offensive map[string]struct{}
var smsConn *sms.Conn
var emailConn *email.Conn
var conf = &PluginJSON{Dependencies: map[string]string{}}
var pluginsGo = []dt.PluginConfig{}
var envLoaded bool

// bClassifiers holds the trained bayesian classifiers for our plugins. The key
// is the plugin ID to which the trained classifier belongs.
var bClassifiers = map[uint64]*bayesian.Classifier{}

// pluginIntents holds the intents for which each plugin has been trained. The
// outer map divides the intents for each plugin by plugin ID.
var pluginIntents = map[uint64][]bayesian.Class{}

// tSentence is a training sentence retrieved from a remote source (defaults to
// itsabot.org). To change the default source, set the ITSABOT_URL environment
// variable.
type tSentence struct {
	ID       uint64
	Sentence string
	Intent   string
	PluginID uint64
}

// Conf returns Abot's plugins.json configuration.
func Conf() *PluginJSON {
	return conf
}

// DB returns a connection the database. This is used internally across
// packages, and isn't needed when building plugins. If you're building a
// plugin, use your plugin's p.DB instead.
func DB() *sqlx.DB {
	return db
}

// NewServer connects to the database and boots all plugins before returning a
// server connection, database connection, and map of offensive words.
func NewServer() (r *httprouter.Router, err error) {
	if err = LoadEnvVars(); err != nil {
		return nil, err
	}
	if len(os.Getenv("ABOT_SECRET")) < 32 && os.Getenv("ABOT_ENV") == "production" {
		return nil, errors.New("must set ABOT_SECRET env var in production to >= 32 characters")
	}
	if db == nil {
		db, err = ConnectDB()
		if err != nil {
			return nil, fmt.Errorf("could not connect to database: %s", err.Error())
		}
	}
	err = LoadConf()
	if err != nil && os.Getenv("ABOT_ENV") != "test" {
		log.Info("failed loading conf", err)
		return nil, err
	}
	if err = checkRequiredEnvVars(); err != nil {
		return nil, err
	}
	err = loadPluginsGo()
	if err != nil && os.Getenv("ABOT_ENV") != "test" {
		log.Info("failed loading plugins.go", err)
		return nil, err
	}
	ner, err = buildClassifier()
	if err != nil {
		log.Debug("could not build classifier", err)
	}
	go func() {
		if os.Getenv("ABOT_ENV") != "test" {
			log.Info("training classifiers")
		}
		if err = trainClassifiers(); err != nil {
			log.Info("could not train classifiers", err)
		}
	}()
	offensive, err = buildOffensiveMap()
	if err != nil {
		log.Debug("could not build offensive map", err)
	}
	p := filepath.Join("assets", "html", "layout.html")
	if err = loadHTMLTemplate(p); err != nil {
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

	// Send any scheduled events on boot and every minute
	evtChan := make(chan *dt.ScheduledEvent)
	go sendEventsTick(evtChan, time.Now())
	go sendEvents(evtChan, 1*time.Minute)

	// Update cached analytics data on boot and every 15 minutes
	go updateAnalyticsTick(time.Now())
	go updateAnalytics(15 * time.Minute)

	return r, nil
}

// compileAssets compresses and merges assets from Abot core and all plugins on
// boot. In development, this step is repeated on each server HTTP request prior
// to serving any assets.
func compileAssets() error {
	p := filepath.Join("cmd", "compileassets.sh")
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
	tmplLayout, err = template.ParseFiles(filepath.Join(
		os.Getenv("ABOT_PATH"), p))
	if tmplLayout == nil {
		tmplLayout, err = template.ParseFiles(p)
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
func LoadConf() error {
	p := filepath.Join(os.Getenv("ABOT_PATH"), "plugins.json")
	okVal := fmt.Sprintf("open %s: no such file or directory", p)
	contents, err := ioutil.ReadFile(p)
	if err != nil && err.Error() != okVal {
		return err
	}
	return json.Unmarshal(contents, conf)
}

// LoadEnvVars from abot.env into memory
func LoadEnvVars() error {
	if envLoaded {
		return nil
	}
	if len(os.Getenv("ABOT_PATH")) == 0 {
		p := filepath.Join(os.Getenv("GOPATH"), "src", "github.com",
			"itsabot", "abot")
		log.Debug("ABOT_PATH not set. defaulting to", p)
		if err := os.Setenv("ABOT_PATH", p); err != nil {
			return err
		}
	}
	if len(os.Getenv("ITSABOT_URL")) == 0 {
		log.Debug("ITSABOT_URL not set, using https://www.itsabot.org")
		err := os.Setenv("ITSABOT_URL", "https://www.itsabot.org")
		if err != nil {
			return err
		}
	}
	p := filepath.Join(os.Getenv("ABOT_PATH"), "abot.env")
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
	envLoaded = true
	return nil
}

// loadPluginsGo loads the plugins.go file into memory.
func loadPluginsGo() error {
	p := filepath.Join(os.Getenv("ABOT_PATH"), "plugins.go")
	okVal := fmt.Sprintf("open %s: no such file or directory", p)
	contents, err := ioutil.ReadFile(p)
	if err != nil && err.Error() != okVal {
		return err
	}
	var val []byte
	var foundStart bool
	var nestLvl int
	for _, b := range contents {
		switch b {
		case '{':
			nestLvl++
			if nestLvl == 1 {
				foundStart = true
			}
		case '}':
			nestLvl--
			if nestLvl == 0 {
				val = append(val, b)
				val = append(val, []byte(",")...)
				foundStart = false
			}
		}
		if !foundStart {
			continue
		}
		val = append(val, b)
	}
	if len(val) == 0 {
		return nil
	}
	val = append([]byte("["), val...)
	val = append(val[:len(val)-1], []byte("]")...)
	return json.Unmarshal(val, &pluginsGo)
}

// trainClassifiers trains classifiers for each plugin.
func trainClassifiers() error {
	for _, pconf := range pluginsGo {
		ss, err := fetchTrainingSentences(pconf.ID, pconf.Name)
		if err != nil {
			return err
		}

		// Assemble list of Bayesian classes from all trained intents
		// for this plugin. m is used to keep track of the classes
		// already taught to each classifier.
		m := map[string]struct{}{}
		for _, s := range ss {
			_, ok := m[s.Intent]
			if ok {
				continue
			}
			log.Debug("learning intent", s.Intent)
			m[s.Intent] = struct{}{}
			pluginIntents[s.PluginID] = append(pluginIntents[s.PluginID],
				bayesian.Class(s.Intent))
		}

		// Build classifier from complete sets of intents
		for _, s := range ss {
			intents := pluginIntents[s.PluginID]
			// Calling bayesian.NewClassifier() with 0 or 1
			// classes causes a panic.
			if len(intents) == 0 {
				break
			}
			if len(intents) == 1 {
				intents = append(intents, bayesian.Class("__no_intent"))
			}
			c := bayesian.NewClassifier(intents...)
			bClassifiers[s.PluginID] = c
		}

		// With classifiers initialized, train each of them on a
		// sentence's stems.
		for _, s := range ss {
			tokens := TokenizeSentence(s.Sentence)
			stems := StemTokens(tokens)
			c, exists := bClassifiers[s.PluginID]
			if exists {
				c.Learn(stems, bayesian.Class(s.Intent))
			}
		}
	}
	return nil
}

// fetchTrainingSentences retrieves training sentences from a remote source
// (via ITSABOT_URL, which defaults to itsabot.org).
func fetchTrainingSentences(pluginID uint64, name string) ([]tSentence, error) {
	c := &http.Client{Timeout: 10 * time.Second}
	pid := strconv.FormatUint(pluginID, 10)
	u := os.Getenv("ITSABOT_URL") + "/api/plugins/train/" + pid
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Info("failed to close response body.", err)
		}
	}()
	ss := []tSentence{}

	// This occurs when the plugin has not been published, which we should
	// ignore on boot.
	if resp.StatusCode == http.StatusBadRequest {
		log.Infof("warn: plugin %s has not been published", name)
		return ss, nil
	}
	err = json.NewDecoder(resp.Body).Decode(&ss)
	return ss, err
}
