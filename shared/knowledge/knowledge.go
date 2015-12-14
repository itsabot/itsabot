// Package knowledge provides known and commonly required information about the
// user to 3rd party apps, such as a user's last known location.
package knowledge

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
)

var ErrNoLocation = errors.New("no previous location")

type Query struct {
	ID        uint64
	UserID    uint64
	Term      string
	Relation  sql.NullString
	WordType  string
	Trigram   string
	Active    bool
	CreatedAt *time.Time
}

// GetLocation returns the last known location of a user. If the location isn't
// recent, ask the user to confirm.
func GetLocation(db *sqlx.DB, u *dt.User) (*dt.Location, string,
	error) {
	var loc *dt.Location
	if u.LocationID == 0 {
		return loc, language.QuestionLocation(""), nil
	}
	q := `
		SELECT name, createdat
		FROM locations
		WHERE userid=$1
		ORDER BY createdat DESC`
	err := db.Get(loc, q, u.ID)
	if err == sql.ErrNoRows {
		return loc, language.QuestionLocation(""), nil
	} else if err != nil {
		return loc, "", err
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	if loc.CreatedAt.Before(yesterday) {
		return loc, language.QuestionLocation(loc.Name), nil
	}
	return loc, "", nil
}

func GetAddress(db *sqlx.DB, u *dt.User, sent string) (*dt.Address, error) {
	var val string
	for _, w := range strings.Fields(sent) {
		if w == "home" || w == "office" {
			val = w
			break
		}
	}
	if len(val) == 0 {
		return nil, nil
	}
	q := `
		SELECT name, line1, line2, city, state, country, zip
		WHERE userid=$1 AND name=$2 AND cardid=0`
	var addr *dt.Address
	if err := db.Get(addr, q, u.ID, val); err != nil {
		return nil, err
	}
	return addr, nil
}

// TODO move Query related things to Ava, making them private
func GetActiveQuery(db *sqlx.DB, u *dt.User) (*Query, error) {
	qry := &Query{}
	q := `SELECT id, userid, term, wordtype, relation
	      FROM knowledgequeries
	      WHERE userid=$1 AND active=TRUE`
	err := db.Select(qry, q, u.ID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return qry, err
}

func (qry *Query) Solve(db *sqlx.DB, m *dt.Msg) error {
	q := `UPDATE knowledgequeries
	      SET relation=$1, active=FALSE
	      WHERE id=$2`
	_, err := db.Exec(q, m.Input.StructuredInput.Objects.String(), qry.ID)
	return err
}

func (qry *Query) Text() string {
	var s string
	if qry.WordType == "Object" {
		s = "What's " + qry.Term + "?"
	} else if qry.WordType == "Command" {
		s = "What do you mean by " + strings.ToLower(qry.Term) + "?"
	}
	return s
}

func SetQueriesInactive(db *sqlx.DB, u *dt.User) error {
	q := `UPDATE knowledgequeries SET active=FALSE WHERE userid=$1`
	_, err := db.Exec(q, u.ID)
	return err
}

// TODO refactor, cleanup
func NewQueries(db *sqlx.DB, p *pkg.Pkg, m *dt.Msg) ([]Query, error) {
	var queries []Query
	queryBase := Query{
		UserID: m.User.ID,
		Active: true,
	}
	var cmd *struct {
		Index int
		Word  string
	}
	var obj *struct {
		Index int
		Word  string
	}
	words := strings.Fields(m.Input.Sentence)
	ssc := m.Input.StructuredInput.Commands.StringSlice()
	for i := len(ssc) - 1; i > 0; i-- {
		// we want the command that's closest to the object, i.e. the
		// last command in the sentence
		c := ssc[i]
		if !p.Vocab.Commands[c] {
			// this can and should be done much more efficiently
			ww := language.RemoveStopWords(words)
			for j := len(ww) - 1; j > 0; j-- {
				if ww[j] != c {
					continue
				}
				cmd = &struct {
					Index int
					Word  string
				}{
					Index: j,
					Word:  c,
				}
			}
		}
		break
	}
	log.Println(m.Input.StructuredInput.Objects.StringSlice())
	for _, w := range m.Input.StructuredInput.Objects.StringSlice() {
		if !p.Vocab.Objects[w] {
			// this can and should be done much more efficiently
			for i, wrd := range language.RemoveStopWords(words) {
				if wrd != w {
					continue
				}
				if obj == nil {
					obj = &struct {
						Index int
						Word  string
					}{
						Index: i,
						Word:  w,
					}
				} else {
					obj.Word += " " + w
				}
			}
		}
	}
	tx, err := db.Beginx()
	if err != nil {
		return queries, err
	}
	if cmd != nil {
		query := queryBase
		query.Term = cmd.Word
		tmp := strings.Fields(m.Input.Sentence)
		tmp = language.RemoveStopWords(tmp)
		query.Trigram = tmp[cmd.Index]
		if len(tmp) > cmd.Index+2 {
			query.Trigram += " " + tmp[cmd.Index+1] + " " +
				tmp[cmd.Index+2]
		} else if len(tmp) > cmd.Index+1 {
			query.Trigram += " " + tmp[cmd.Index+1]
		}
		query.WordType = "Command"
		q := `INSERT INTO knowledgequeries
		      (term, trigram, wordtype, userid) VALUES ($1, $2, $3, $4)
		      RETURNING id`
		err := tx.QueryRowx(q, query.Term, query.Trigram,
			query.WordType, m.User.ID).Scan(&query.ID)
		if err != nil {
			return queries, err
		}
		queries = append(queries, query)
	}
	if obj != nil {
		query := queryBase
		query.Term = obj.Word
		tmp := strings.Fields(m.Input.Sentence)
		tmp = language.RemoveStopWords(tmp)
		query.Trigram = tmp[obj.Index]
		if len(tmp) > obj.Index+2 {
			query.Trigram += " " + tmp[obj.Index+1] + " " +
				tmp[obj.Index+2]
		} else if len(tmp) > obj.Index+1 {
			query.Trigram += " " + tmp[obj.Index+1]
		}
		query.WordType = "Object"
		q := `INSERT INTO knowledgequeries
		      (term, trigram, wordtype, userid) VALUES ($1, $2, $3, $4)
		      RETURNING id`
		err := tx.QueryRowx(q, query.Term, query.Trigram,
			query.WordType, m.User.ID).Scan(&query.ID)
		if err != nil {
			return queries, err
		}
		queries = append(queries, query)
	}
	err = tx.Commit()
	return queries, err
}

func Search(db *sqlx.DB) []Query {
}
