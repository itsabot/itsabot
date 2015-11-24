package dt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

type jsonState json.RawMessage

type Msg struct {
	User         *User
	Input        *Input
	LastResponse *Resp
	Route        string
}

// ResponseMsg is used to pass results from packages to Ava
type RespMsg struct {
	ResponseID int
	Sentence   string
}

type Resp struct {
	ID         int
	UserID     int
	InputID    int
	FeedbackID int
	Sentence   string
	Route      string
	State      map[string]interface{}
	CreatedAt  time.Time
}

type Feedback struct {
	Id        int
	Sentence  string
	Sentiment int
	CreatedAt time.Time
}

const (
	SentimentNegative = -1
	SentimentNeutral  = 0
	SentimentPositive = 1
)

type Input struct {
	ID                int
	UserID            int
	FlexID            string
	FlexIDType        int
	Sentence          string
	SentenceAnnotated string
	ResponseID        int
	StructuredInput   *StructuredInput
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

func (m *Msg) GetLastResponse(db *sqlx.DB) error {
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
	m.LastResponse = &Resp{
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

func (m *Msg) NewResponse() *Resp {
	var uid int
	if m.User != nil {
		uid = m.User.ID
	}
	res := &Resp{
		UserID:  uid,
		InputID: m.Input.ID,
		Route:   m.Route,
	}
	if m.LastResponse != nil {
		res.State = m.LastResponse.State
	}
	return res
}
