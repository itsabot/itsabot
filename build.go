package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

type packagesConf struct {
	Name         string
	Version      string
	Dependencies map[string]string
}

// bootDependencies executes all binaries listed in "packages.json". each
// dependencies is passed the rpc address of the ava core. it is expected that
// each dependency respond with there own rpc address when registering
// themselves with the ava core
func bootDependencies(avaRPCAddr string) {
	log.WithFields(log.Fields{
		"ava_core_addr": avaRPCAddr,
	}).Debugln("booting dependencies")
	content, err := ioutil.ReadFile("packages.json")
	if err != nil {
		log.Fatalln("reading packages.json", err)
	}
	var conf packagesConf
	if err := json.Unmarshal(content, &conf); err != nil {
		log.Fatalln("err: unmarshaling packages", err)
	}
	for name := range conf.Dependencies {
		log.WithFields(log.Fields{"pkg": name}).Debugln("booting")
		// NOTE assumes packages are installed with go install ./...,
		// matching Heroku's Go buildpack
		cmd := exec.Command(name, "-coreaddr", avaRPCAddr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			log.WithFields(log.Fields{
				"pkg": name,
			}).Fatalln(err)
		}
	}
}
