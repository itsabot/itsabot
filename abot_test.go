package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/core/log"
)

func TestMain(m *testing.M) {
	if err := core.LoadEnvVars(); err != nil {
		log.Info("failed to load env vars", err)
	}
	if err := os.Setenv("ABOT_ENV", "test"); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestPluginSearch(t *testing.T) {
	query := "weather"
	var byt []byte
	var err error
	if testing.Short() {
		log.Info("stubbing plugin search results in short mode.")
		byt = []byte(`[{"Name":"Weather","Valid":true}]`)
	} else {
		byt, err = searchItsAbot(query)
		if err != nil {
			t.Fatal(err)
		}
	}
	var b []byte
	buf := bytes.NewBuffer(b)
	if err = outputPluginResults(buf, byt); err != nil {
		t.Fatal(err)
	}
	tmp := buf.String()
	if !strings.Contains(tmp, "NAME") {
		t.Fatal(err)
	}
	if !strings.Contains(tmp, "Weather") {
		t.Fatal(err)
	}
}

func TestPluginGenerate(t *testing.T) {
	defer func() {
		if err := os.RemoveAll("__test"); err != nil {
			log.Info("failed to clean up __test dir.", err)
		}
		if err := os.RemoveAll("test_here"); err != nil {
			log.Info("failed to clean up __test dir.", err)
		}
	}()

	// generate plugin
	l := log.New("")
	l.SetFlags(0)
	if err := generatePlugin(l, "__test"); err != nil {
		t.Fatal(err)
	}
	if err := generatePlugin(l, "TestHere"); err != nil {
		t.Fatal(err)
	}
}
