package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"errors"
	"os"
	"path"
	"strings"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jbrukh/bayesian"
	"github.com/avabot/ava/shared/nlp"
)

var (
	ErrMissingFlexIdType = errors.New("missing flexidtype")
)

func train(c *bayesian.Classifier, s string) error {
	log.Infoln("training classifier")
	if err := trainClassifier(c, s); err != nil {
		return err
	}
	buf := bytes.NewBuffer([]byte{})
	if err := c.WriteTo(buf); err != nil {
		return err
	}
	q := `UPDATE ml SET data=$1 WHERE name='ner'`
	_, err := db.Exec(q, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func loadClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	log.Debugln("loading classifier")
	var err error
	q := `SELECT data FROM ml WHERE name='ner' LIMIT 1`
	row := db.QueryRowx(q)
	var tmp []byte
	err = row.Scan(&tmp)
	if err == sql.ErrNoRows {
		c, err = buildClassifier(c)
		if err != nil {
			log.WithField("fn", "loadClassifier").Errorln(err)
			return c, err
		}
	} else if err != nil {
		log.WithField("fn", "loadClassifier").Errorln(err)
		return c, err
	}
	buf := bytes.NewBuffer(tmp)
	c, err = bayesian.NewClassifierFromReader(buf)
	if err != nil {
		log.WithField("fn", "loadClassifier").Errorln(err)
		return c, err
	}
	log.Infoln("loaded NER classifier")
	return c, nil
}

func buildClassifier(c *bayesian.Classifier) (*bayesian.Classifier, error) {
	c = bayesian.NewClassifier(nlp.Command, nlp.Actor, nlp.Object, nlp.Time,
		nlp.Place, nlp.None)
	filename := path.Join("data", "training", "imperative.txt")
	fi, err := os.Open(filename)
	if err != nil {
		log.WithField("fn", "buildClassifier").Errorln(err)
		return c, err
	}
	defer fi.Close()
	scanner := bufio.NewScanner(fi)
	line := 1
	for scanner.Scan() {
		if err := trainClassifier(c, scanner.Text()); err != nil {
			log.WithFields(log.Fields{
				"fn":   "buildClasssifier",
				"line": line,
			}).Errorln(err)
		}
		line++
	}
	if err := scanner.Err(); err != nil {
		log.WithField("fn", "buildClassifier").Errorln(err)
		return c, err
	}
	buf := bytes.NewBuffer([]byte{})
	if err := c.WriteTo(buf); err != nil {
		log.WithField("fn", "buildClassifier").Errorln(err)
		return c, err
	}
	q := `INSERT INTO ml (name, data) VALUES ('ner', $1)`
	_, err = db.Exec(q, buf.Bytes())
	if err != nil {
		log.WithField("fn", "buildClassifier").Errorln(err)
		return c, err
	}
	log.Infoln("new classifier trained")
	return c, nil
}

func trainClassifier(c *bayesian.Classifier, s string) error {
	if len(s) == 0 || s[0] == '/' {
		return nil
	}
	ws := strings.Fields(s)
	l := len(ws)
	for i := 0; i < l; i++ {
		var word2 string
		var word3 string
		word1, entity, err := nlp.ExtractEntity(ws[i])
		if err != nil {
			return err
		}
		if entity == "" {
			continue
		}
		trigram := word1
		if i+1 < l {
			word2, _, err = nlp.ExtractEntity(ws[i+1])
			if err != nil {
				return err
			}
			trigram += " " + word2
		}
		if i+2 < l {
			word3, _, err = nlp.ExtractEntity(ws[i+2])
			if err != nil {
				return err
			}
			trigram += " " + word3
		}
		c.Learn([]string{word1}, entity)
		if word2 != "" {
			c.Learn([]string{word1 + " " + word2}, entity)
		}
		if word3 != "" {
			c.Learn([]string{trigram}, entity)
		}
	}
	return nil
}
