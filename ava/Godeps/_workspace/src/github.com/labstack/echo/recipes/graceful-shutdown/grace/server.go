package main

import (
	"net/http"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/labstack/echo" // Setup
	"github.com/facebookgo/grace/gracehttp"
)

func main() {

	e := echo.New()
	e.Get("/", func(c *echo.Context) error {
		return c.String(http.StatusOK, "Six sick bricks tick")
	})

	gracehttp.Serve(e.Server(":1323"))
}
