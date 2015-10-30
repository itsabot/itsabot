package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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
	if err := loadMT(); err != nil {
		return err
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

func approveAssignment(foreignID, assignmentID string) error {
	if err := loadMT(); err != nil {
		return err
	}
	if len(assignmentID) == 0 || assignmentID == "ASSIGNMENT_ID_NOT_AVAILABLE" {
		// Assignment was completed outside of MTurk
		err := expireHIT(foreignID)
		return err
	}
	var resp interface{}
	service := "AWSMechanicalTurkRequester"
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	operation := "ApproveAssignment"
	params := make(map[string]string)
	params["AssignmentId"] = assignmentID
	params["Operation"] = operation
	params["RequesterFeedback"] = "Thank you!"
	sign(mt.Auth, service, operation, timestamp, params)
	url := *mt.URL // make a copy
	url.RawQuery = multimap(params).Encode()
	r, err := http.Get(url.String())
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		errCode := fmt.Sprintf("%d: unexpected status code", r.StatusCode)
		return errors.New(errCode)
	}
	dec := xml.NewDecoder(r.Body)
	err = dec.Decode(resp)
	r.Body.Close()
	if err != nil {
		return err
	}
	log.Printf("ApproveAssignment: %+v\n", resp)
	return nil
}

func expireHIT(foreignID string) error {
	var resp interface{}
	service := "AWSMechanicalTurkRequester"
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	operation := "ForceExpireHIT"
	params := make(map[string]string)
	params["HITId"] = foreignID
	params["Operation"] = operation
	sign(mt.Auth, service, operation, timestamp, params)
	url := *mt.URL // make a copy
	url.RawQuery = multimap(params).Encode()
	r, err := http.Get(url.String())
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		errCode := fmt.Sprintf("%d: unexpected status code", r.StatusCode)
		return errors.New(errCode)
	}
	dec := xml.NewDecoder(r.Body)
	err = dec.Decode(&resp)
	r.Body.Close()
	if err != nil {
		return err
	}
	log.Printf("ForceExpireHIT: %+v\n", resp)
	return nil
}

func sign(auth aws.Auth, service, method, timestamp string, params map[string]string) {
	params["AWSAccessKeyId"] = mt.Auth.AccessKey
	params["Service"] = service
	params["Timestamp"] = timestamp
	payload := service + method + timestamp
	hash := hmac.New(sha1.New, []byte(auth.SecretKey))
	hash.Write([]byte(payload))
	signature := make([]byte, base64.StdEncoding.EncodedLen(hash.Size()))
	base64.StdEncoding.Encode(signature, hash.Sum(nil))
	params["Signature"] = string(signature)
}

func multimap(p map[string]string) url.Values {
	q := make(url.Values, len(p))
	for k, v := range p {
		q[k] = []string{v}
	}
	return q
}

func loadMT() error {
	auth, err := aws.EnvAuth()
	if err != nil {
		return err
	}
	if mt == nil {
		if os.Getenv("AVA_ENV") == "production" {
			mt = mturk.New(auth, false)
		} else {
			mt = mturk.New(auth, true)
		}
	}
	return nil
}
