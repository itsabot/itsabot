package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"

	log "github.com/Sirupsen/logrus"
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
		log.Error("reading packages.json", err)
		log.Fatal("could not start")
	}
	var conf packagesConf
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Error("unmarshaling packages", err)
		log.Fatal("could not start")
	}
	i := 1
	for name := range conf.Dependencies {
		p := strconv.Itoa(4000 + i)
		plog := log.WithFields(log.Fields{
			"port":    p,
			"package": name,
		})
		plog.Debug("booting package")
		pth := path.Join("ava_modules", name, name)
		cmd := exec.Command(pth, "-port", p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			plog.Fatal("running package process: ", err)
		}
		i += 2
	}
}
