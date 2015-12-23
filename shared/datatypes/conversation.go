package dt

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/dchest/stemmer/porter2"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
)

var (
	ErrMissingUser = errors.New("missing user")
)

type jsonState json.RawMessage

type Msg struct {
	User         *User
	Input        *Input
	LastInput    *Input
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
	// SentenceFields breaks the sentence into words. Tokens like ,.' are
	// treated as individual words.
	SentenceFields  []string
	ResponseID      uint64
	KnowledgeFilled bool
	StructuredInput *StructuredInput
}

func (in *Input) Save(db *sqlx.DB) error {
	q := `UPDATE inputs SET knowledgefilled=TRUE WHERE id=$1`
	_, err := db.Exec(q, in.ID)
	return err
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

func NewMessage(db *sqlx.DB, u *User, in *Input) *Msg {
	words := strings.Fields(in.Sentence)
	eng := porter2.Stemmer
	stems := []string{}
	for _, w := range words {
		w = strings.TrimRight(w, ",.?;:!-/")
		stems = append(stems, eng.Stem(w))
	}
	var err error
	m := &Msg{User: u, Input: in, Stems: stems}
	m, err = addContext(db, m)
	if err != nil {
		log.WithField("fn", "addContext").Errorln(err)
	}
	return m
}

func (m *Msg) GetLastInput(db *sqlx.DB) error {
	log.Debugln("getting last input")
	q := `SELECT id, sentence, knowledgefilled FROM inputs
	      WHERE userid=$1
	      ORDER BY createdat DESC`
	var tmp Input
	if err := db.Get(&tmp, q, m.User.ID); err != nil {
		return err
	}
	tmp.SentenceFields = SentenceFields(tmp.Sentence)
	m.LastInput = &tmp
	return nil
}

func (m *Msg) GetLastResponse(db *sqlx.DB) error {
	log.Debugln("getting last response")
	if m.User == nil {
		return ErrMissingUser
	}
	q := `SELECT id, stateid, route, sentence, userid
	      FROM responses
	      WHERE userid=$1
	      ORDER BY createdat DESC`
	row := db.QueryRowx(q, m.User.ID)
	var tmp struct {
		ID       uint64
		Route    string
		Sentence string
		StateID  sql.NullInt64
		UserID   uint64
	}
	log.Debugln("scanning into response")
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
	m.LastResponse = &Resp{
		ID:       tmp.ID,
		Route:    tmp.Route,
		Sentence: tmp.Sentence,
		UserID:   tmp.UserID,
	}
	var state []byte
	q = `SELECT state FROM states WHERE id=$1`
	err = db.Get(&state, q, tmp.StateID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	err = json.Unmarshal(state, &m.LastResponse.State)
	if err != nil {
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

// addContext to a StructuredInput, replacing pronouns with the nouns to which
// they refer. TODO refactor
func addContext(db *sqlx.DB, m *Msg) (*Msg, error) {
	for _, w := range m.Input.StructuredInput.Pronouns() {
		var ctx string
		var err error
		switch Pronouns[w] {
		case ObjectI:
			ctx, err = getContextObject(db, m.User,
				m.Input.StructuredInput,
				"objects")
			if err != nil {
				return m, err
			}
			if ctx == "" {
				return m, nil
			}
			for i, o := range m.Input.StructuredInput.Objects {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Objects[i] = ctx
			}
		case ActorI:
			ctx, err = getContextObject(db, m.User,
				m.Input.StructuredInput, "actors")
			if err != nil {
				return m, err
			}
			if ctx == "" {
				return m, nil
			}
			for i, o := range m.Input.StructuredInput.Actors {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Actors[i] = ctx
			}
		case TimeI:
			ctx, err = getContextObject(db, m.User,
				m.Input.StructuredInput, "times")
			if err != nil {
				return m, err
			}
			if ctx == "" {
				return m, nil
			}
			for i, o := range m.Input.StructuredInput.Times {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Times[i] = ctx
			}
		case PlaceI:
			ctx, err = getContextObject(db, m.User,
				m.Input.StructuredInput, "places")
			if err != nil {
				return m, err
			}
			if ctx == "" {
				return m, nil
			}
			for i, o := range m.Input.StructuredInput.Places {
				if o != w {
					continue
				}
				m.Input.StructuredInput.Places[i] = ctx
			}
		default:
			return m, errors.New("unknown type found for pronoun")
		}
		log.WithFields(log.Fields{
			"fn":  "addContext",
			"ctx": ctx,
		}).Infoln("context found")
	}
	return m, nil
}

func getContextObject(db *sqlx.DB, u *User, si *StructuredInput,
	datatype string) (string, error) {
	log.Debugln("getting object context")
	var tmp *StringSlice
	if u == nil {
		return "", ErrMissingUser
	}
	if u != nil {
		q := `
			SELECT ` + datatype + `
			FROM inputs
			WHERE userid=$1 AND array_length(objects, 1) > 0`
		if err := db.Get(&tmp, q, u.ID); err != nil {
			return "", err
		}
	}
	return tmp.Last(), nil
}

func SentenceFields(s string) []string {
	var ret []string
	for _, w := range strings.Fields(s) {
		var end bool
		for _, r := range w {
			switch r {
			case '\'', '"', ',', '.', ':', ';', '!', '?':
				end = true
				ret = append(ret, string(r))
			}
		}
		if end {
			ret = append(ret, strings.ToLower(w[:len(w)-1]))
		} else {
			ret = append(ret, strings.ToLower(w))
		}
	}
	return ret
}
