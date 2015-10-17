package datatypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

type jsonState json.RawMessage

type Message struct {
	User         *User
	Input        *Input
	LastResponse *Response
}

type Response struct {
	ID        int
	UserID    int
	InputID   int
	Sentence  string
	State     map[string]interface{} //jsonState
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
		log.Error("unmarshal jsonState: ", err)
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
		SELECT state
		FROM responses
		WHERE userid=$1
		ORDER BY createdat DESC`
	if m.User == nil {
		// TODO move to shared errors
		return errors.New("missing user")
	}
	row := db.QueryRowx(q, m.User.ID)
	if err := row.StructScan(m.LastResponse); err != nil {
		log.Error("ERROR DB: ", err)
		return err
	}
	return nil
}

func (r *Response) QuestionLanguage() bool {
	if r.Sentence == "Where are you now?" ||
		r.Sentence[0:17] == "Are you still in " {
		return true
	}
	return false
}

func NewResponse(m *Message) *Response {
	return &Response{
		UserID:  m.User.ID,
		InputID: m.Input.ID,
	}
}
