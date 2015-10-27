package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/AdRoll/goamz/exp/mturk"
)

var mt *mturk.MTurk

func aidedTrain(trainID int) error {
	var categorizationMasterQualID string
	if os.Getenv("AVA_ENV") == "production" {
		mt = mturk.New(auth, false)
		categorizationMasterQualID = "2NDP2L92HECWY8NS8H3CK0CP5L9GHO"
	} else {
		mt = mturk.New(auth, true)
		categorizationMasterQualID = "2F1KVCNHMVHV8E9PBUB2A4J79LU20F"
	}
	title := "Identify elements of a sentence."
	desc := "Find and identify commands, objects, actors, times, and places in a sentence."
	qxn := &ExternalQuestion{
		ExternalURL: os.Getenv("BASE_URL") + "train",
		FrameHeight: 700,
	}
	reward := Price{
		Amount:       0.03,
		CurrencyCode: "USD",
	}
	timelimitInSeconds := 300
	lifetimeInSeconds := 31536000 // 365 days
	keywords := "ava,machine,learning,language,speech,english,train"
	maxAssignments := 1
	qualReq := &QualificationRequirement{
		QualificationTypeId: categorizationMasterQualID,
		Comparator:          "Exists",
		RequiredToPreview:   true,
	}
	annotation := strconv.Itoa(trainID)
	hit, err := mt.CreateHIT(title, desc, qxn, reward,
		timelimitInSeconds, lifetimeInSeconds, keywords, maxAssignments,
		qualReq, annotation)
	if err != nil {
		return err
	}
	hit.HITTypeId
	return nil
}

// cronTrain runs every few minutes, checking HIT statuses and training the
// bayes classifier when each is complete.
func cronTrain() error {
	var t struct {
		ID        int
		ForeignID string
	}
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
	a, err := mt.GetAssignmentsForHITResponse(foreignID)
	if err != nil {
		return err
	}
	log.Println("mturk answers", a.Answers())
	return nil
}
