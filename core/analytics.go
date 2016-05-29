package core

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/itsabot/abot/core/log"
)

// Keys used in the database for types of analytics.
const (
	keyUserCount  = "userCount"
	keyMsgCount   = "msgCount"
	keyTrainCount = "trainCount"
	keyVersion    = "version"
)

// updateAnalytics recursively calls itself to continue running.
func updateAnalytics(interval time.Duration) {
	t := time.NewTicker(time.Hour)
	select {
	case now := <-t.C:
		updateAnalyticsTick(now)
		updateAnalytics(interval)
	}
}

func updateAnalyticsTick(t time.Time) {
	if os.Getenv("ABOT_ENV") == "test" {
		return
	}
	log.Info("updating analytics")
	createdAt := t.Round(24 * time.Hour)

	// User count
	var count int
	q := `SELECT COUNT(*) FROM (
		SELECT DISTINCT (flexid, flexidtype) FROM userflexids
	      ) AS t`
	if err := db.Get(&count, q); err != nil {
		log.Info("failed to retrieve user count.", err)
		return
	}
	aq := `INSERT INTO analytics (label, value, createdat)
	       VALUES ($1, $2, $3)
	       ON CONFLICT (label, createdat) DO UPDATE SET value=$2`
	_, err := db.Exec(aq, keyUserCount, count, createdAt)
	if err != nil {
		log.Info("failed to update analytics (user count).", err)
		return
	}

	// Message count
	q = `SELECT COUNT(*) FROM messages`
	if err = db.Get(&count, q); err != nil {
		log.Info("failed to retrieve message count.", err)
		return
	}
	_, err = db.Exec(aq, keyMsgCount, count, createdAt)
	if err != nil {
		log.Info("failed to update analytics (msg count).", err)
		return
	}

	// Messages needing training
	q = `SELECT COUNT(*) FROM messages
	     WHERE needstraining=TRUE AND abotsent=FALSE`
	if err = db.Get(&count, q); err != nil {
		log.Info("failed to retrieve user count.", err)
		return
	}
	_, err = db.Exec(aq, keyTrainCount, count, createdAt)
	if err != nil {
		log.Info("failed to update analytics (msg count).", err)
		return
	}

	// Version number
	client := &http.Client{Timeout: 15 * time.Minute}
	u := "https://raw.githubusercontent.com/egtann/abot/master/plugins.json"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Info("failed to retrieve version number.", err)
		return
	}
	reqResp, err := client.Do(req)
	if err != nil {
		log.Info("failed to retrieve version number.", err)
		return
	}
	defer func() {
		if err = reqResp.Body.Close(); err != nil {
			log.Info("failed to close body.", err)
		}
	}()
	var remoteConf PluginJSON
	if err = json.NewDecoder(reqResp.Body).Decode(&remoteConf); err != nil {
		log.Info("failed to retrieve version number.", err)
		return
	}
	_, err = db.Exec(aq, keyVersion, remoteConf.Version, createdAt)
	if err != nil {
		log.Info("failed to update analytics (version number).", err)
		return
	}
}
