package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
)

type packagesConf struct {
	Name         string
	Version      string
	Dependencies map[string]string
}

// TODO Fetches packages via package manager, puts them in a dir. ava_modules

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
	i := 2
	for name := range conf.Dependencies {
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			log.Fatalln("PORT must be an integer")
		}
		p := strconv.Itoa(port + i)
		log.Println("booting package", name, p)
		pth := path.Join("ava_modules", name, name)
		cmd := exec.Command(pth, "-port", p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			log.Fatalln("running package process: ", err)
		}
		i += 2
	}
}
