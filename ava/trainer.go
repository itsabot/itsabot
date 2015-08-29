package main

import (
	"io/ioutil"
	"log"
	"path"
	"strings"

	"github.com/jbrukh/bayesian"
	_ "github.com/lib/pq"
)

// trainedClassifier will load data from the DB if available. Otherwise it
// trains on the local data in the /training folder.
func trainedClassifier(bc map[string]bayesian.Class) (*bayesian.Classifier, error) {
	c, err := newTrainedClassifier(bc)
	return c, err
	/*
		read := []byte{}
		reader := bytes.NewReader(read)
		err := db.Get(&reader, "SELECT data FROM training")
		log.Println("READER", reader)

			if err == sql.ErrNoRows {
			log.Println("no classifier data found in DB. Creating new classifier.")
			buf := []byte{}
			b := bytes.NewBuffer(buf)
			err := c.WriteTo(b)
			if err != nil {
				log.Fatalln("couldn't write classifier to buffer", err)
			}

			_, err = db.Exec("INSERT INTO training (data) VALUES ($1)", b.Bytes())
			if err != nil {
				log.Fatalln("couldn't save classifer to DB", err)
			}
		} else {
			c, err = bayesian.NewClassifierFromReader(reader)
		}
		return c, err
	*/
}

// newTrainedClassifier trains on the local data in the /training folder.
// There's no reason to call this function directly. It's instead used as a
// fallback within trainedClassifier() if no trained classifier can be found.
func newTrainedClassifier(bc map[string]bayesian.Class) (*bayesian.Classifier, error) {
	var vals []bayesian.Class
	for _, v := range bc {
		vals = append(vals, v)
	}
	c := bayesian.NewClassifier(vals...)

	files, err := ioutil.ReadDir("training")
	if err != nil {
		return c, err
	}

	for _, file := range files {
		log.Println("reading", file.Name())
		d, err := ioutil.ReadFile(path.Join("training", file.Name()))
		if err != nil {
			log.Fatalln("error reading file", file.Name(), err)
		}
		data := strings.Split(string(d), " ")
		cat := bc[strings.TrimSuffix(file.Name(), ".train")]
		log.Println("data", data)
		log.Println("cat", cat)
		c.Learn(data, cat)
	}
	return c, err
}
