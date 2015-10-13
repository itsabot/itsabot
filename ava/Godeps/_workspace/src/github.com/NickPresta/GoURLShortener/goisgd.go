// Library to shorten URIs using http://is.gd
package goisgd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Shortens the URI using the "API" listed here: http://is.gd/api_info.php
func Shorten(uri string) (string, error) {
	u := "http://is.gd/api.php?longurl=" + url.QueryEscape(uri)

	response, err := http.Get(u)

	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	shortUri := string(body)

	// Make sure we get a 200 response code, otherwise,
	// return the error message returned by is.gd
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(shortUri)
	}

	return shortUri, nil
}
