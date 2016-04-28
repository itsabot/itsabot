package dt

import (
	"database/sql"
	"time"

	"github.com/itsabot/abot/shared/nlp"
	"github.com/jmoiron/sqlx"
)

// Msg is a message received by a user. It holds various fields that are useful
// for plugins which are populated by Abot core in core/process.
type Msg struct {
	ID              uint64
	FlexID          string
	FlexIDType      FlexIDType
	Sentence        string
	User            *User
	StructuredInput *nlp.StructuredInput
	Stems           []string
	Plugin          string
	State           map[string]interface{}
	CreatedAt       *time.Time
	// AbotSent determines if msg is from the user or Abot
	AbotSent      bool
	NeedsTraining bool
	// Tokens breaks the sentence into words. Tokens like ,.' are treated as
	// individual words.
	Tokens []string
	Route  string
}

// GetMsg returns a message for a given message ID.
func GetMsg(db *sqlx.DB, id uint64) (*Msg, error) {
	q := `SELECT id, sentence, abotsent
	      FROM messages
	      WHERE id=$1`
	m := &Msg{}
	if err := db.Get(m, q, id); err != nil {
		return nil, err
	}
	return m, nil
}

// Update a message as needing training.
func (m *Msg) Update(db *sqlx.DB) error {
	q := `UPDATE messages
	      SET needstraining=$1
	      WHERE id=$2`
	if _, err := db.Exec(q, m.NeedsTraining, m.ID); err != nil {
		return err
	}
	return nil
}

// Save a message to the database, updating the message ID.
func (m *Msg) Save(db *sqlx.DB) error {
	q := `INSERT INTO messages
	      (userid, sentence, plugin, route, abotsent, needstraining, flexid,
		flexidtype)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	row := db.QueryRowx(q, m.User.ID, m.Sentence, m.Plugin, m.Route,
		m.AbotSent, m.NeedsTraining, m.User.FlexID, m.User.FlexIDType)
	if err := row.Scan(&m.ID); err != nil {
		return err
	}
	return nil
}

// GetLastRoute for a given user so the previous plugin can be called again if
// no new trigger is detected.
func (m *Msg) GetLastRoute(db *sqlx.DB) (string, error) {
	var route string
	q := `SELECT route FROM messages
	      WHERE userid=$1 AND abotsent IS FALSE
	      ORDER BY createdat DESC`
	err := db.Get(&route, q, m.User.ID)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	return route, nil
}
