// Package util provides utility functions for Abot's core and shared packages.
package util

import (
	"fmt"
	"net/url"

	"github.com/labstack/echo"
)

// CookieVal retrieves a cookie's value from an HTTP echo request as a string.
func CookieVal(c *echo.Context, name string) (value string, err error) {
	ck, err := c.Request().Cookie(name)
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, name)
	}
	val, err := url.QueryUnescape(ck.Value)
	if err != nil {
		return "", err
	}
	return val, nil
}
