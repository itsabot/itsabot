package dt

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/dchest/stemmer/porter2"
)

type jsonState json.RawMessage

type Msg struct {
	User         *User
	Input        *Input
	LastResponse *Resp
	Stems        []string
	Route        string
}

// ResponseMsg is used to pass results from packages to Ava
type RespMsg struct {
	ResponseID uint64
	Sentence   string
}

type Resp struct {
	ID         uint64
	UserID     uint64
	InputID    uint64
	FeedbackID uint64
	Sentence   string
	Route      string
	State      map[string]interface{}
	CreatedAt  time.Time
}

type Feedback struct {
	Id        uint64
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
	ID                uint64
	UserID            uint64
	FlexID            string
	FlexIDType        int
	Sentence          string
	SentenceNorm      string
	SentenceAnnotated string
	ResponseID        uint64
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

func NewInput(si *StructuredInput, uid uint64, fid string, fidT int) *Input {
	in := Input{
		StructuredInput: si,
		UserID:          uid,
		FlexID:          fid,
		FlexIDType:      fidT,
	}
	return &in
}

func NewMessage(u *User, in *Input) *Msg {
	words := strings.Fields(in.Sentence)
	eng := porter2.Stemmer
	stems := []string{}
	for _, w := range words {
		w = strings.TrimRight(w, ",.?;:!-/")
		stems = append(stems, eng.Stem(w))
	}
	return &Msg{User: u, Input: in, Stems: stems}
}

func (m *Msg) GetLastResponse(db *sqlx.DB) error {
	if m.User == nil {
		return errors.New("missing user")
	}
	q := `SELECT stateid, route, sentence, userid
	      FROM responses
	      WHERE userid=$1
	      ORDER BY createdat DESC`
	row := db.QueryRowx(q, m.User.ID)
	var tmp struct {
		Route    string
		Sentence string
		StateID  sql.NullInt64
		UserID   uint64
	}
	err := row.StructScan(&tmp)
	if err == sql.ErrNoRows {
		m.LastResponse = &Resp{}
		return nil
	}
	if err != nil {
		log.Println("structscan row ", err)
		return err
	}
	if !tmp.StateID.Valid {
		return errors.New("invalid stateid")
	}
	var state []byte
	q = `SELECT state FROM states WHERE id=$1`
	if err = db.Get(&state, q, tmp.StateID); err != nil {
		return err
	}
	m.LastResponse = &Resp{
		Route:    tmp.Route,
		Sentence: tmp.Sentence,
		UserID:   tmp.UserID,
	}
	if err = json.Unmarshal(state, &m.LastResponse.State); err != nil {
		log.Println("unmarshaling state", err)
		return err
	}
	return nil
}

func (m *Msg) NewResponse() *Resp {
	var uid uint64
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
