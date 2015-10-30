package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
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
		log.Println("err: reading packages.json", err)
		log.Fatalln("could not start")
	}
	var conf packagesConf
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Println("err: unmarshaling packages", err)
		log.Fatalln("could not start")
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalln("PORT must be an integer")
	}
	i := 2
	for name := range conf.Dependencies {
		p := strconv.Itoa(port + i)
		log.Println("booting package", name, p)
		// NOTE assumes packages are installed with go install ./...,
		// matching Heroku's Go buildpack
		cmd := exec.Command(name, "-port", p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			log.Fatalln("running package process: ", err)
		}
		i += 2
	}
}
