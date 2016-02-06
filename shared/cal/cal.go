// Package cal enables control over Gmail calendars
package cal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/avabot/ava/Godeps/_workspace/src/golang.org/x/oauth2"
	"github.com/avabot/ava/Godeps/_workspace/src/golang.org/x/oauth2/google"
	"github.com/jmoiron/sqlx"
	"golang.org/x/net/context"
)

// config is the configuration specification supplied to the OAuth package.
var config = &oauth2.Config{
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	// Scope determines which API calls you are authorized to make
	Scopes:   []string{"https://www.googleapis.com/auth/calendar"},
	Endpoint: google.Endpoint,
	// Use "postmessage" for the code-flow for server side apps
	RedirectURL: "postmessage",
}

// Token represents an OAuth token response.
type Token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IdToken     string `json:"id_token"`
}

// ClaimSet represents an IdToken response.
type ClaimSet struct {
	Sub string
}

// Exchange takes an authentication code and exchanges it with the OAuth
// endpoint for a Google API bearer token and a Google+ ID
func Exchange(code string) (accessToken string, idToken string, err error) {
	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return "", "", fmt.Errorf("Error while exchanging code: %v", err)
	}
	// TODO: return ID token in second parameter from updated oauth2 interface
	return tok.AccessToken, tok.Extra("id_token").(string), nil
}

// DecodeIdToken takes an ID Token and decodes it to fetch the Google+ ID within
func DecodeIdToken(idToken string) (gID string, err error) {
	// An ID token is a cryptographically-signed JSON object encoded in base 64.
	// Normally, it is critical that you validate an ID token before you use it,
	// but since you are communicating directly with Google over an
	// intermediary-free HTTPS channel and using your Client Secret to
	// authenticate yourself to Google, you can be confident that the token you
	// receive really comes from Google and is valid. If your server passes the ID
	// token to other components of your app, it is extremely important that the
	// other components validate the token before using it.
	var set ClaimSet
	if idToken != "" {
		// Check that the padding is correct for a base64decode
		parts := strings.Split(idToken, ".")
		if len(parts) < 2 {
			return "", fmt.Errorf("Malformed ID token")
		}
		// Decode the ID token
		b, err := base64Decode(parts[1])
		if err != nil {
			return "", fmt.Errorf("Malformed ID token: %v", err)
		}
		err = json.Unmarshal(b, &set)
		if err != nil {
			return "", fmt.Errorf("Malformed ID token: %v", err)
		}
	}
	return set.Sub, nil
}

func base64Decode(s string) ([]byte, error) {
	// add back missing padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func Client(db *sqlx.DB, uid uint64) (*http.Client, error) {
	context := context.Background()
	var token string
	q := `SELECT token FROM sessions WHERE userid=$1 AND label='gcal_token'`
	if err := db.Get(&token, q, uid); err != nil {
		return nil, err
	}
	t := oauth2.Token{}
	t.AccessToken = token
	return config.Client(context, &t), nil
}

func Events(client *http.Client) error {
	srv, err := calendar.New(client)
	if err != nil {
		return err
	}
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		return err
	}
	if len(events.Items) > 0 {
		for _, i := range events.Items {
			var when string
			// If the DateTime is an empty string the Event is an all-day Event.
			// So only Date is available.
			if i.Start.DateTime != "" {
				when = i.Start.DateTime
			} else {
				when = i.Start.Date
			}
			fmt.Printf("%s (%s)\n", i.Summary, when)
		}
	} else {
		fmt.Printf("No upcoming events found.\n")
	}
	return nil
}
