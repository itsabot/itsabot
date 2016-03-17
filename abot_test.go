package main

import (
	"bytes"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/log"
	"github.com/labstack/echo"
)

var e *echo.Echo
var phone *string

func TestMain(m *testing.M) {
	phone = flag.String("phone", "+13105555555", "phone number of test user")
	flag.Parse()
	if err := os.Setenv("ABOT_ENV", "test"); err != nil {
		os.Exit(1)
	}
	var err error
	log.Info("starting server")
	e, _, _, err = core.NewServer()
	if err != nil {
		log.Info("failed to start server", err)
		os.Exit(1)
	}
	log.SetDebug(true)
	os.Exit(m.Run())
}

func TestIndex(t *testing.T) {
	u := "http://localhost:" + os.Getenv("PORT")
	c, b := request("GET", u, nil, e)
	if c != http.StatusOK {
		log.Debug(string(b))
		t.Fatal("expected", http.StatusOK, "got", c)
	}
}

func TestSignupSubmit(t *testing.T) {
	reset(t)
	u := "http://localhost:" + os.Getenv("PORT")
	base := u + "/api/signup.json"
	data := []byte(`{
		"Name": "Tester",
		"Email": "test@example.com",
		"Password": "password",
		"FID": "+13105555555"
	}`)
	c, b := request("POST", base, data, e)
	if c != http.StatusOK {
		log.Debug(string(b))
		t.Fatal("expected", http.StatusOK, "got", c)
	}
}

func request(method, path string, data []byte, e *echo.Echo) (int, string) {
	r, err := http.NewRequest(method, path, bytes.NewBuffer(data))
	r.Header.Add("Content-Type", "application/json")
	if err != nil {
		return 0, "err completing request: " + err.Error()
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func reset(t *testing.T) {
	_, err := core.DB().Exec(`DELETE FROM users`)
	if err != nil {
		t.Fatal(err)
	}
}
