package datatypes

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

type jsonState json.RawMessage

type Message struct {
	User  *User
	Input *Input
}

type Response struct {
	Id        int
	UserId    int
	InputId   int
	Response  string
	State     jsonState
	CreatedAt time.Time
}

type Input struct {
	UserId          int
	FlexId          string
	FlexIdType      int
	Sentence        string
	ResponseId      int
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
		UserId:          uid,
		FlexId:          fid,
		FlexIdType:      fidT,
	}
	return &in
}

func (m *Message) LastResponse(db *sqlx.DB, r *Response) error {
	q := `
		SELECT state
		FROM responses
		WHERE userid=$1
		ORDER BY createdat DESC`
	if m.User == nil {
		// TODO move to shared errors
		return errors.New("missing user")
	}
	row := db.QueryRowx(q, m.User.Id)
	if err := row.StructScan(r); err != nil {
		log.Error("ERROR DB: ", err)
		return err
	}
	return nil
}
