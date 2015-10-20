package datatypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

type jsonState json.RawMessage

type Message struct {
	User         *User
	Input        *Input
	LastResponse *Response
	Route        string
}

// ResponseMsg is used to pass results from packages to Ava
type ResponseMsg struct {
	ResponseID int
	Sentence   string
}

type Response struct {
	ID        int
	UserID    int
	InputID   int
	Sentence  string
	Route     string
	State     map[string]interface{}
	CreatedAt time.Time
}

type Input struct {
	ID              int
	UserID          int
	FlexID          string
	FlexIDType      int
	Sentence        string
	ResponseID      int
	StructuredInput *StructuredInput
}

func (j *jsonState) Scan(value interface{}) error {
	if err := json.Unmarshal(value.([]byte), *j); err != nil {
		log.Println("unmarshal jsonState: ", err)
		return err
	}
	return nil
}

func (j *jsonState) Value() (driver.Value, error) {
	return j, nil
}

func NewInput(si *StructuredInput, uid int, fid string, fidT int) *Input {
	in := Input{
		StructuredInput: si,
		UserID:          uid,
		FlexID:          fid,
		FlexIDType:      fidT,
	}
	return &in
}

func (m *Message) GetLastResponse(db *sqlx.DB) error {
	q := `
		SELECT state, route, sentence, userid
		FROM responses
		WHERE userid=$1
		ORDER BY createdat DESC`
	if m.User == nil {
		// TODO move to shared errors
		return errors.New("missing user")
	}
	row := db.QueryRowx(q, m.User.ID)
	var tmp struct {
		State    []byte
		Route    string
		Sentence string
		UserID   int
	}
	if err := row.StructScan(&tmp); err != nil {
		log.Println("structscan row ", err)
		return err
	}
	m.LastResponse = &Response{
		Route:    tmp.Route,
		Sentence: tmp.Sentence,
		UserID:   tmp.UserID,
	}
	if err := json.Unmarshal(tmp.State, &m.LastResponse.State); err != nil {
		log.Println("unmarshaling state", err)
		return err
	}
	return nil
}

func (m *Message) NewResponse() *Response {
	var uid int
	if m.User != nil {
		uid = m.User.ID
	}
	res := &Response{
		UserID:  uid,
		InputID: m.Input.ID,
		Route:   m.Route,
	}
	if m.LastResponse != nil {
		res.State = m.LastResponse.State
	}
	return res
}
