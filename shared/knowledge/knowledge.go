// Package knowledge provides known and commonly required information about the
// user to 3rd party apps, such as a user's last known location.
package knowledge

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/dchest/stemmer/porter2"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
)

var ErrNoLocation = errors.New("no previous location")

type Query struct {
	ID         uint64
	UserID     uint64
	ResponseID sql.NullInt64
	Term       string
	Relation   sql.NullString
	WordType   string
	Trigram    string
	Active     bool
	CreatedAt  *time.Time
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

func LastQueryResponseID(db *sqlx.DB, m *dt.Msg) (uint64, error) {
	var rid sql.NullInt64
	q := `SELECT responseid FROM knowledgequeries
	      WHERE userid=$1
	      ORDER BY createdat DESC`
	err := db.Get(&rid, q, m.User.ID)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return uint64(rid.Int64), nil
}

func SolveLastQuery(db *sqlx.DB, m *dt.Msg) error {
	q := `UPDATE knowledgequeries
	      SET relation=$1, active=FALSE, responseid=$2
	      WHERE userid=$3 AND active=TRUE AND term<>$4`
	s := m.Input.StructuredInput.Objects.String()
	res, err := db.Exec(q, s, m.LastResponse.ID, m.User.ID, s)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
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

func DeleteQueries(db *sqlx.DB, u *dt.User) error {
	q := `DELETE FROM knowledgequeries WHERE userid=$1 AND relation IS NULL`
	if _, err := db.Exec(q, u.ID); err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

// TODO refactor, cleanup
// Ava asks questions about words she doesn't understand. To do that,
// she creates a knowledge query, which is stored separately from last
// responses. The knowledge table can be thought of like a graph, where
// words are related to one another within degrees of separation. From
// most specific to less and less specific. Questions are asked
// recursively up to two times. For example:
//
// User> Buy me a sau gri.
// Ava>  What's a sau gri?
// User> Sauvignon Gris
// Ava>  (passes "Buy me a Sauvignon Gris" to the package, but is still
//        confused)
// Ava>  And what's a Sauvignon Gris?
// User> white wine
// Ava>  (searches for Buy me a Sauvignon Gris white wine, which the
//        purchase package can handle)
//
// Then Ava knows that sau gri is very similar to "Sauvignon Gris", and
// somewhat similar to "white wine". The next time a user asks for a
// "sau gri" in the same or similar trigram context, "buy me sau gri",
// she'll jump right to "Buy me a Sauvignon Gris white wine".
//
// Ava should also handle confusion across multiple wordtypes in
// sequence. For example:
//
// User> Procure libations
// Ava>  What do you mean by procure?
// User> buy
// Ava>  Ok. And what do you mean by libations?
// User> booze
// Ava>  What do you mean by booze?
// User> alcohol
// Ava>  (pass "Buy alcohol" to the package)
func extractCommand(p *pkg.Pkg, m *dt.Msg, words,
	ssc []string) *language.WordT {
	var cmd *language.WordT
	eng := porter2.Stemmer
	for i := len(ssc) - 1; i > 0; i-- {
		// we want the command that's closest to the object, i.e. the
		// last command in the sentence
		c := ssc[i]
		if !p.Vocab.Commands[c] {
			// this can and should be done much more efficiently
			ww := language.RemoveStopWords(words)
			for j := len(ww) - 1; j > 0; j-- {
				ww[j] = eng.Stem(ww[j])
				if ww[j] != c {
					continue
				}
				cmd = &language.WordT{
					Index: j,
					Word:  c,
				}
			}
		}
		break
	}
	return cmd
}

func extractObject(p *pkg.Pkg, m *dt.Msg, words, ssc []string) *language.WordT {
	var obj *language.WordT
	seen := map[string]bool{}
	for _, w := range m.Input.StructuredInput.Objects.StringSlice() {
		log.Debugln("word", w)
		if !p.Vocab.Objects[w] {
			log.Debugln("here...")
			// this can and should be done much more efficiently
			for i, wrd := range language.RemoveStopWords(words) {
				log.Debugln("here!", w, wrd)
				if wrd != w {
					continue
				}
				log.Debugln("here!!!!", i, wrd)
				if obj == nil {
					obj = &language.WordT{
						Index: i,
						Word:  w,
					}
					log.Debugln(i, w)
				} else if !seen[w] {
					obj.Word += " " + w
					log.Debugln(i, w)
				}
				seen[w] = true
			}
		}
	}
	return obj
}

func buildQuery(wt *language.WordT, m *dt.Msg, wordType string,
	words []string) Query {
	query := Query{
		UserID: m.User.ID,
		Active: true,
	}
	if wordType == "Command" {
		query.Term = strings.ToLower(wt.Word)
	} else {
		query.Term = wt.Word
	}
	tmp := strings.Fields(m.Input.Sentence)
	tmp = language.RemoveStopWords(tmp)
	query.Trigram = tmp[wt.Index]
	if len(tmp) > wt.Index+2 {
		query.Trigram += " " + tmp[wt.Index+1] + " " +
			tmp[wt.Index+2]
	} else if len(tmp) > wt.Index+1 {
		query.Trigram += " " + tmp[wt.Index+1]
	}
	query.Trigram = strings.ToLower(query.Trigram)
	query.WordType = wordType
	if m.LastResponse != nil {
		query.ResponseID.Int64 = int64(m.LastResponse.ID)
		query.ResponseID.Valid = true
	}
	return query
}

func NewQueriesForPkg(db *sqlx.DB, p *pkg.Pkg, m *dt.Msg) ([]Query, error) {
	log.Debugln("creating new knowledge queries")
	words := strings.Fields(m.Input.Sentence)
	ssc := m.Input.StructuredInput.Commands.StringSlice()
	sso := m.Input.StructuredInput.Objects.StringSlice()
	log.Debugln("sso", sso)
	cmd := extractCommand(p, m, words, ssc)
	obj := extractObject(p, m, words, sso)
	var queries []Query
	if cmd != nil {
		log.Debugln("building knowledge query around command")
		query := buildQuery(cmd, m, "Command", words)
		if err := getOrCreateKnowledgeQuery(db, &query); err != nil {
			return queries, err
		}
		queries = append(queries, query)
	}
	if obj != nil {
		log.Debugln("building knowledge query around object")
		log.Debugln("obj", obj.Word)
		query := buildQuery(obj, m, "Object", words)
		if err := getOrCreateKnowledgeQuery(db, &query); err != nil {
			return queries, err
		}
		queries = append(queries, query)
	}
	log.Debugln("built", len(queries), "queries")
	return queries, nil
}

func FillIn(db *sqlx.DB, objs []string, sent string, u *dt.User) (string, bool,
	error) {
	l := len(objs)
	log.Debugln("original length", l)
	for i := range objs {
		if i+2 > l {
			break
		}
		objs = append(objs, objs[i]+" "+objs[i+1])
		log.Debugln("appending bigram", i, objs)
		if i+3 > l {
			continue
		}
		log.Debugln(i+2, "<=", l)
		objs = append(objs, objs[i]+" "+objs[i+1]+" "+objs[i+2])
		log.Debugln("appending trigram", i, objs)
	}
	// TODO perform this search recursively twice, e.g.
	// sau gri -> Sauvignon Gris
	// Sauvignon Gris -> white wine
	q := `SELECT term, relation FROM knowledgequeries
	      WHERE userid=? AND term IN (?)
	      ORDER BY termlength DESC`
	var changed bool
	q, args, err := sqlx.In(q, u.ID, objs)
	if err != nil {
		return sent, changed, err
	}
	q = db.Rebind(q)
	var query Query
	err = db.Get(&query, q, args...)
	if err != nil && err != sql.ErrNoRows {
		return sent, changed, err
	}
	words := strings.Fields(strings.ToLower(sent))
	for i := range words {
		words[i] = strings.TrimRight(words[i], ".,:;-/?!'\"")
	}
Loop:
	for i, w := range words {
		for _, qt := range strings.Fields(query.Term) {
			log.Debugln("here 2", w, qt, query.Relation.Valid)
			if w != qt || !query.Relation.Valid {
				continue
			}
			words[i] = query.Relation.String + " " + words[i]
			changed = true
			break Loop
		}
	}
	return strings.Join(words, " "), changed, nil
}

func getOrCreateKnowledgeQuery(db *sqlx.DB, query *Query) error {
	q := `SELECT id FROM knowledgequeries WHERE
	      term=$1 AND userid=$2 AND trigram=$3`
	err := db.Get(&query.ID, q, query.Term, query.UserID, query.Trigram)
	if err == sql.ErrNoRows {
		log.Debugln("inserting knowledgequery with responseid",
			query.ResponseID)
		q = `INSERT INTO knowledgequeries
	             (term, trigram, wordtype, userid, responseid, termlength)
		     VALUES ($1, $2, $3, $4, $5, $6)
	             RETURNING id`
		err = db.QueryRowx(q, query.Term, query.Trigram,
			query.WordType, query.UserID, query.ResponseID,
			len(query.Term)).Scan(&query.ID)
		if err != nil {
			return err
		}
	} else {
		log.Debugln("getOrCreateKnowledgeQuery", "row found", err)
	}
	log.Debugf("%+v\n", *query)
	return err
}
