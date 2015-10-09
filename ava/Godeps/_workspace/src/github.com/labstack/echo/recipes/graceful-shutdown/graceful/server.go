package main

import (
	"net/http"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo"
	"github.com/tylerb/graceful"
)

func main() {
	// Setup
	e := echo.New()
	e.Get("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "Sue sews rose on slow jor crows nose")
	})

	graceful.ListenAndServe(e.Server(":1323"), 5*time.Second)
}
