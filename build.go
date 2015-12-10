package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

type packagesConf struct {
	Name         string
	Version      string
	Dependencies map[string]string
}

func bootDependencies() {
	// TODO Inspect for errors
	content, err := ioutil.ReadFile("packages.json")
	if err != nil {
		log.Fatalln("reading packages.json", err)
	}
	var conf packagesConf
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Fatalln("err: unmarshaling packages", err)
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalln("PORT must be an integer")
	}
	i := 2
	for name := range conf.Dependencies {
		p := strconv.Itoa(port + i)
		log.WithFields(log.Fields{
			"pkg":  name,
			"port": p,
		}).Debugln("booting")
		// NOTE assumes packages are installed with go install ./...,
		// matching Heroku's Go buildpack
		cmd := exec.Command(name, "-port", p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			log.WithFields(log.Fields{
				"pkg":  name,
				"port": p,
			}).Fatalln(err)
		}
		i += 2
	}
}
