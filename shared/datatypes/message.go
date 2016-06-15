package dt

import (
	"database/sql"
	"time"

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
	StructuredInput *StructuredInput
	Stems           []string
	Plugin          *Plugin
	CreatedAt       *time.Time
	// AbotSent determines if msg is from the user or Abot
	AbotSent      bool
	NeedsTraining bool
	Trained       bool
	// Tokens breaks the sentence into words. Tokens like ,.' are treated as
	// individual words.
	Tokens []string
	Route  string

	Usage []string
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
	q := `UPDATE messages SET needstraining=$1 WHERE id=$2`
	if _, err := db.Exec(q, m.NeedsTraining, m.ID); err != nil {
		return err
	}
	return nil
}

// Save a message to the database, updating the message ID.
func (m *Msg) Save(db *sqlx.DB) error {
	var pluginName string
	if m.Plugin != nil {
		pluginName = m.Plugin.Config.Name
	}
	q := `INSERT INTO messages
	      (userid, sentence, plugin, route, abotsent, needstraining, flexid,
		flexidtype, trained)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	row := db.QueryRowx(q, m.User.ID, m.Sentence, pluginName, m.Route,
		m.AbotSent, m.NeedsTraining, m.User.FlexID, m.User.FlexIDType,
		m.Trained)
	if err := row.Scan(&m.ID); err != nil {
		return err
	}
	return nil
}

// GetLastPlugin for a given user so the previous plugin can be called again if
// no new trigger is detected.
func (m *Msg) GetLastPlugin(db *sqlx.DB) (string, string, error) {
	var res struct {
		Plugin string
		Route  string
	}
	var err error
	if m.User.ID > 0 {
		q := `SELECT route, plugin FROM messages
		      WHERE userid=$1 AND abotsent IS FALSE
		      ORDER BY createdat DESC`
		err = db.Get(&res, q, m.User.ID)
	} else {
		q := `SELECT route, plugin FROM messages
		      WHERE flexid=$1 AND flexidtype=$2 AND abotsent IS FALSE
		      ORDER BY createdat DESC`
		err = db.Get(&res, q, m.User.FlexID, m.User.FlexIDType)
	}
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}
	return res.Plugin, res.Route, nil
}
