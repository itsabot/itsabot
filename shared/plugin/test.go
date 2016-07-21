package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/itsabot/abot/core"
	"github.com/julienschmidt/httprouter"
)

// TestPrepare sets the ABOT_ENV, cleans out the test DB, and initializes the
// server for testing.
func TestPrepare() *httprouter.Router {
	if err := os.Setenv("ABOT_ENV", "test"); err != nil {
		log.Fatal(err)
	}
	r, err := core.NewServer()
	if err != nil {
		log.Fatal("failed to start abot server.", err)
	}
	TestCleanup()
	return r
}

// TestReq tests the input against the expected output. It returns an error if
// the input does not match the expected output.
func TestReq(r *httprouter.Router, in string, exp []string) error {
	data := struct {
		FlexIDType int
		FlexID     string
		CMD        string
	}{
		FlexIDType: 3,
		FlexID:     "0",
		CMD:        in,
	}
	byt, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal req. %s", err)
	}
	c, b := request(r, "POST", os.Getenv("ABOT_URL")+"/", byt)
	if c != http.StatusOK {
		return fmt.Errorf("expected %d, got %d. %s", http.StatusOK, c, b)
	}
	var found bool
	for _, x := range exp {
		if strings.Contains(b, x) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("expected %s, got %q for %q", exp, b, in)
	}
	return nil
}

// TestCleanup cleans up the test DB. It can only be run when ABOT_ENV ==
// "test".
func TestCleanup() {
	if os.Getenv("ABOT_ENV") != "test" {
		log.Fatal(`TestCleanup() can only be run when ABOT_ENV == "test"`)
	}
	q := `DELETE FROM users`
	_, err := core.DB().Exec(q)
	if err != nil {
		log.Print("failed to delete users.", err)
	}
	q = `DELETE FROM messages`
	_, err = core.DB().Exec(q)
	if err != nil {
		log.Print("failed to delete messages.", err)
	}
	q = `DELETE FROM states`
	_, err = core.DB().Exec(q)
	if err != nil {
		log.Print("failed to delete states.", err)
	}
}

func request(r *httprouter.Router, method, path string, data []byte) (int,
	string) {

	req, err := http.NewRequest(method, path, bytes.NewBuffer(data))
	if err != nil {
		return 0, "err completing request: " + err.Error()
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, string(w.Body.Bytes())
}
