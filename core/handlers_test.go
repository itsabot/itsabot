package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/julienschmidt/httprouter"
)

var router *httprouter.Router

func TestMain(m *testing.M) {
	if err := os.Setenv("ABOT_ENV", "test"); err != nil {
		os.Exit(1)
	}
	var err error
	router, err = NewServer()
	if err != nil {
		log.Info("failed to start server", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestHMain(t *testing.T) {
	reset(t)
	user, fid, fidT := seedDBUser(t)
	req := dt.Request{CMD: "Hi", UserID: user.ID}

	// Test via a UserID
	byt, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	c, b := request("POST", "/", byt)
	if c != http.StatusOK {
		log.Info(b)
		t.Fatal("expected", http.StatusOK, "got", c)
	}
	if b == "Something went wrong with my wiring... I'll get that fixed up soon." {
		t.Fatal(`expected "Hi there :)" but got "Something went wrong..."`)
	}

	// Test via a FlexID
	req.UserID = 0
	req.FlexID = fid
	req.FlexIDType = fidT
	byt, err = json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	c, b = request("POST", "/", byt)
	if c != http.StatusOK {
		log.Info(b)
		t.Fatal("expected", http.StatusOK, "got", c)
	}
}

func TestHIndex(t *testing.T) {
	c, b := request("GET", "/", nil)
	if c != http.StatusOK {
		log.Info(b)
		t.Fatal("expected", http.StatusOK, "got", c)
	}
}

func TestHAPILoginSubmit(t *testing.T) {
	reset(t)
	user, _, _ := seedDBUser(t)
	data := struct {
		Email    string
		Password string
	}{
		Email:    user.Email,
		Password: user.Password,
	}
	byt, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	u := "http://localhost:" + os.Getenv("PORT") + "/api/login.json"
	c, b := request("POST", u, byt)
	if c != http.StatusOK {
		log.Info(b)
		t.Fatal("expected", http.StatusOK, "got", c)
	}
}

func TestHSignupSubmit(t *testing.T) {
	reset(t)
	u := "http://localhost:" + os.Getenv("PORT") + "/api/signup.json"
	data := []byte(`{
		"Name": "Tester",
		"Email": "test@example.com",
		"Password": "password",
		"FID": "+13105555555"
	}`)
	c, b := request("POST", u, data)
	if c != http.StatusOK {
		log.Info(b)
		t.Fatal("expected", http.StatusOK, "got", c)
	}
}

func TestHAPILogoutSubmit(t *testing.T) {
	reset(t)
	user, _, _ := seedDBUser(t)
	seedDBUserSession(t, user)

	// Make request with cookie
	router := newRouter()
	u := "http://localhost:" + os.Getenv("PORT") + "/api/logout.json"
	r, err := http.NewRequest("POST", u, nil)
	r.Header.Add("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{
		Name:  "id",
		Value: strconv.Itoa(int(user.ID)),
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
}

func TestHAPIProfile(t *testing.T) {
	reset(t)
	user, _, _ := seedDBUser(t)
	seedDBUserSession(t, user)

	c, b := userRequest("GET", "/api/user/profile.json", nil, user)
	if c != http.StatusOK {
		log.Info(b)
		t.Fatal("expected", http.StatusOK, "got", c)
	}
	if !strings.Contains(b, "Name") {
		t.Fatal(`expected "Name" but got`, b)
	}
}

func request(method, path string, data []byte) (int, string) {
	router := newRouter()
	u := "http://localhost:" + os.Getenv("PORT")
	u += path
	r, err := http.NewRequest(method, u, bytes.NewBuffer(data))
	if err != nil {
		return 0, "error completing request: " + err.Error()
	}
	r.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func userRequest(method, path string, data []byte, user *dt.User) (int,
	string) {

	router := newRouter()
	u := "http://localhost:" + os.Getenv("PORT")
	u += path
	r, err := http.NewRequest(method, u, bytes.NewBuffer(data))
	r.Header.Add("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{
		Name:  "id",
		Value: strconv.Itoa(int(user.ID)),
	})
	if err != nil {
		return 0, "error completing request: " + err.Error()
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func reset(t *testing.T) {
	_, err := db.Exec(`DELETE FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`DELETE FROM sessions`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`DELETE FROM messages`)
	if err != nil {
		t.Fatal(err)
	}
}

func seedDBUser(t *testing.T) (u *dt.User, fid string, fidT dt.FlexIDType) {
	u = &dt.User{
		Name:     "t",
		Email:    "t@example.com",
		Password: "password",
	}
	q := `INSERT INTO users (name, email, password, locationid)
	      VALUES ($1, $2, $3, 0)
	      RETURNING id`
	row := db.QueryRowx(q, u.Name, u.Email, u.Password)
	var uid uint64
	if err := row.Scan(&uid); err != nil {
		t.Fatal(err)
	}
	u.ID = uid

	fid = "+13105555555"
	fidT = dt.FlexIDType(2)
	q = `INSERT INTO userflexids (flexid, flexidtype, userid)
	     VALUES ($1, $2, $3)`
	if _, err := db.Exec(q, fid, fidT, uid); err != nil {
		t.Fatal(err)
	}
	return u, fid, fidT
}

func seedDBUserSession(t *testing.T, u *dt.User) {
	q := `INSERT INTO sessions (userid, token) VALUES ($1, '')`
	_, err := db.Exec(q, u.ID)
	if err != nil {
		t.Fatal(err)
	}
}
