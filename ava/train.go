package main

import (
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
		Amount:       "0.05",
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
	if err = updateTraining(trainID, hit.HITId); err != nil {
		return err
	}
	return nil
}
