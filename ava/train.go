package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/goamz/goamz/aws"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/goamz/goamz/exp/mturk"
)

var mt *mturk.MTurk

func supervisedTrain(s string) error {
	trainID, err := saveTrainingSentence(s)
	if err != nil {
		return err
	}
	if err = aidedTrain(trainID); err != nil {
		return err
	}
	return nil
}

func aidedTrain(trainID int) error {
	auth, err := aws.EnvAuth()
	if err != nil {
		return err
	}
	if os.Getenv("AVA_ENV") == "production" {
		mt = mturk.New(auth, false)
	} else {
		mt = mturk.New(auth, true)
	}
	title := "Identify elements of a sentence."
	desc := "Find and identify commands, objects, actors, times, and places in a sentence."
	annotation := strconv.Itoa(trainID)
	qxn := &mturk.ExternalQuestion{
		ExternalURL: os.Getenv("BASE_URL") + "?/train/" + annotation,
		FrameHeight: 889,
	}
	reward := mturk.Price{
		Amount:       "0.03",
		CurrencyCode: "USD",
	}
	timelimitInSeconds := uint(300)
	lifetimeInSeconds := uint(31536000) // 365 days
	keywords := "ava,machine,learning,language,speech,english,train"
	maxAssignments := uint(1)
	hit, err := mt.CreateHIT(title, desc, qxn, reward,
		timelimitInSeconds, lifetimeInSeconds, keywords, maxAssignments,
		nil, annotation)
	if err != nil {
		return err
	}
	log.Printf("HIT %+v\n", hit)
	return nil
}

// cronTrain runs every few minutes, checking HIT statuses and training the
// bayes classifier when each is complete.
func cronTrain() error {
	// No LIMIT here, since that could create a queue, which would go
	// unnoticed/need monitoring. Instead, allow the requests to pile up and
	// overload memory, which monitoring services will catch and alert that
	// something needs to change -- likely the price of the MTurk HIT. And
	// since we scan over the rows, it's unlikely without a HUGE amount of
	// traffic to cause a problem. Something left for another time/dev :)
	q := `
		SELECT id, foreignid
		FROM trainings
		WHERE trained=FALSE
		ORDER BY createdat DESC`
	rows, err := db.Queryx(q)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	for rows.Next() {
		var id int
		var foreignID string
		err = rows.Scan(&id, &foreignID)
		if err != nil {
			rows.Close()
			return err
		}
		err = trainTask(id, foreignID)
		if err != nil {
			rows.Close()
			return err
		}
	}
	return nil
}

func trainTask(id int, foreignID string) error {
	a, err := mt.GetAssignmentsForHIT(foreignID)
	if err != nil {
		return err
	}
	log.Println("mturk answers", a.Answers())
	return nil
}
