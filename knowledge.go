package main

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	log "github.com/avabot/ava/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/dchest/stemmer/porter2"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/language"
)

type nodeTermT int

const (
	nodeTermCommand nodeTermT = iota + 1
	nodeTermObject
)

var ErrRelEqTerm = errors.New("rel cannot == term")

type graphObj interface {
	IncrementConfidence(*sqlx.DB) error
	DecrementConfidence(*sqlx.DB) error
	Trm() string
	Rel() string
}

// gsMax defines the max number of edges and nodes returned from a search.
const gsMax int64 = 3

// TODO add trigram support around the term
type node struct {
	ID         uint64
	UserID     uint64
	Term       string
	TermStem   string
	TermLength int
	TermType   nodeTermT
	Confidence int
	Relation   sql.NullString
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

// TODO add support for startnodeterm
type edge struct {
	ID            uint64
	StartNodeID   uint64
	EndNodeID     uint64
	UserID        uint64
	NodePath      dt.Uint64_Slice
	StartNodeTerm string
	Content       string
	Confidence    int
	CreatedAt     *time.Time
}

func searchEdgesForTerm(term string) ([]*edge, error) {
	var edges []*edge
	q := `SELECT startnodeterm, startnodeid, endnodeid, userid, nodepath,
	          confidence
	      FROM knowledgeedges
	      WHERE startnodeterm=$1
	      ORDER BY confidence DESC
	      LIMIT ` + strconv.FormatInt(gsMax, 10)
	if err := db.Select(&edges, q, term); err != nil {
		return nil, err
	}
	return edges, nil
}

func searchEdges(start *node) ([]*edge, error) {
	var edges []*edge
	// consider searching to prioritize one's own userid
	q := `SELECT startnodeterm, startnodeid, endnodeid, userid, nodepath,
	          confidence
	      FROM knowledgeedges
	      WHERE startnodeid=$1
	      ORDER BY confidence DESC
	      LIMIT ` + strconv.FormatInt(gsMax, 10)
	if err := db.Select(&edges, q, start.ID); err != nil {
		return edges, nil
	}
	return edges, nil
}

// searchNodes runs when searchEdges fails to return at least gsMax results
func searchNodes(term string, edgesFound int64) ([]*node, error) {
	var nodes []*node
	bigrams := upToBigrams(term)
	log.Debugln("bigrams", bigrams)
	for _, bg := range bigrams {
		log.Debugln("searching nodes for", bg)
		q := `SELECT id, term, termlength, termtype, relation, userid,
		          createdat
		      FROM knowledgenodes
		      WHERE termstem=$1
		      ORDER BY confidence
		      LIMIT ` + strconv.FormatInt(gsMax-edgesFound, 10)
		err := db.Select(&nodes, q, bg)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if len(nodes) > 0 {
			break
		}
	}
	return nodes, nil
}

func (n *node) updateRelation(db *sqlx.DB, si *dt.StructuredInput) error {
	var rel string
	if n.TermType == nodeTermObject {
		rel = si.Objects.String()
	} else if n.TermType == nodeTermCommand {
		rel = si.Commands.Last()
	} else {
		return errors.New("invalid node TermType")
	}
	if rel == n.Term {
		return ErrRelEqTerm
	}
	q := `UPDATE knowledgenodes SET relation=$1 WHERE id=$2`
	n.Relation.String = rel
	n.Relation.Valid = true
	_, err := db.Exec(q, rel, n.ID)
	log.Debugln("updated relation with", rel, "for id", n.ID)
	return err
}

func (n *node) IncrementConfidence(db *sqlx.DB) error {
	n.Confidence++
	q := `UPDATE knowledgenodes SET confidence=confidence+1 WHERE id=$2`
	_, err := db.Exec(q, n.Confidence, n.ID)
	return err
}

func (n *node) DecrementConfidence(db *sqlx.DB) error {
	n.Confidence--
	q := `UPDATE knowledgenodes SET confidence=confidence-1 WHERE id=$2`
	_, err := db.Exec(q, n.Confidence, n.ID)
	return err
}

func (n *node) Trm() string {
	return n.Term
}

func (n *node) Rel() string {
	log.Debugf("%+v\n", n)
	return n.Relation.String
}

func (n *node) Text() string {
	var s string
	if n.TermType == nodeTermObject {
		s = "What's " + n.Term + "?"
	} else if n.TermType == nodeTermCommand {
		s = "What do you mean by " + strings.ToLower(n.Term) + "?"
	}
	return s
}

func (e *edge) IncrementConfidence(db *sqlx.DB) error {
	e.Confidence++
	q := `UPDATE knowledgenodes SET confidence=confidence+1 WHERE id=$2`
	_, err := db.Exec(q, e.Confidence, e.ID)
	return err
}

func (e *edge) DecrementConfidence(db *sqlx.DB) error {
	e.Confidence--
	q := `UPDATE knowledgeedges SET confidence=confidence-1 WHERE id=$2`
	_, err := db.Exec(q, e.Confidence, e.ID)
	return err
}

func (e *edge) Trm() string {
	return e.StartNodeTerm
}

func (e *edge) Rel() string {
	return e.Content
}

// upToBigrams splits a sentence into all possible 1-gram and 2-gram combos and
// stems each of the terms
func upToBigrams(s string) []string {
	eng := porter2.Stemmer
	s = eng.Stem(s)
	ss := strings.Fields(s)
	for i := range ss {
		if i+1 >= len(ss) {
			break
		}
		ss = append(ss, ss[i]+" "+ss[i+1])
	}
	return ss
}

func replaceSentence(db *sqlx.DB, msg *dt.Msg, g graphObj) (string, error) {
	msg.LastInput = msg.Input
	tmp := []string{}
	// TODO move comparison to stemmed versions
	trm := strings.Fields(g.Trm())
	for i := 0; i < len(msg.LastInput.SentenceFields); i++ {
		if i+len(trm) > len(msg.LastInput.SentenceFields) {
			sf := msg.LastInput.SentenceFields
			tmp = append(tmp, sf[i:len(sf)]...)
			log.Debugln("reached end. stopping")
			break
		}
		wSlice := msg.LastInput.SentenceFields[i : i+len(trm)]
		ws := strings.Join(wSlice, " ")
		log.Debugln("comparing", ws, "=?", trm)
		if ws == g.Trm() {
			log.Debugln("matched. inserting", g.Rel())
			tmp = append(tmp, g.Rel())
		}
		tmp = append(tmp, wSlice[0])
		log.Debugln("tmp", tmp)
	}
	s := updateSentence(tmp)
	log.Debugln("updated sentence", s)
	return s, nil
}

func updateSentence(sentenceFields []string) string {
	if len(sentenceFields) == 0 {
		return ""
	}
	// almost certainly could be done in 1 loop instead of 2, avoiding the
	// separate strings.Join
	var tmp string
	log.Debugln("sentenceFields", sentenceFields)
	for i := 1; i < len(sentenceFields); i++ {
		tmp = sentenceFields[i]
		switch tmp {
		case "\"", "'", ",", ".", ":", ";", "!", "?":
			log.Debugln("found punctuation")
			sentenceFields[i-1] = sentenceFields[i-1] + tmp
			sentenceFields[i] = ""
		}
	}
	return strings.Join(sentenceFields, " ")
}

func getActiveNode(db *sqlx.DB, u *dt.User) (*node, error) {
	n := &node{}
	q := `SELECT id, userid, term, termstem, termtype, relation
	      FROM knowledgenodes
	      WHERE userid=$1 AND relation IS NULL OR relation=''`
	err := db.Get(n, q, u.ID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	log.Debugln("found active node")
	return n, err
}

func newNodes(db *sqlx.DB, av dt.AtomicMap, m *dt.Msg) ([]*node, error) {
	log.Debugln("creating new knowledge nodes")
	words := strings.Fields(m.Input.Sentence)
	if usesOffensiveLanguage(words) {
		return nil, errors.New("I'm sorry, but I don't respond to foul language.")
	}
	ssc := m.Input.StructuredInput.Commands.StringSlice()
	sso := m.Input.StructuredInput.Objects.StringSlice()
	log.Debugln("sso", sso)
	cmd := extractCommand(av, m, words, ssc)
	obj := extractObject(av, m, words, sso)
	eng := porter2.Stemmer
	var nodes []*node
	q := `INSERT INTO knowledgenodes
	      (term, termstem, termlength, termtype, userid)
	      VALUES ($1, $2, $3, $4, $5)
	      RETURNING id`
	var id uint64
	if cmd != nil {
		log.Debugln("building knowledge node around command")
		stem := eng.Stem(cmd.Word)
		n := node{
			Term:       strings.ToLower(cmd.Word),
			TermStem:   stem,
			TermLength: len(cmd.Word),
			TermType:   nodeTermCommand,
		}
		err := db.QueryRowx(q, n.Term, n.TermStem, n.TermLength,
			nodeTermCommand, m.User.ID).Scan(&id)
		if err != nil && err.Error() != `pq: duplicate key value violates unique constraint "knowledgeedges_userid_term_key"` {
			return nodes, err
		}
		n.ID = id
		nodes = append(nodes, &n)
	}
	if obj != nil {
		log.Debugln("building knowledge node around object")
		stem := eng.Stem(obj.Word)
		n := node{
			Term:       strings.ToLower(obj.Word),
			TermStem:   stem,
			TermLength: len(obj.Word),
			TermType:   nodeTermObject,
		}
		err := db.QueryRowx(q, n.Term, n.TermStem, n.TermLength,
			nodeTermObject, m.User.ID).Scan(&id)
		if err != nil && err.Error() != `pq: duplicate key value violates unique constraint "knowledgenodes_userid_term_key"` {
			return nodes, err
		}
		n.ID = id
		nodes = append(nodes, &n)
	}
	log.Debugln("built", len(nodes), "knowledge nodes")
	return nodes, nil
}

func usesOffensiveLanguage(words []string) bool {
	for _, w := range words {
		if language.SwearWords[w] {
			return true
		}
	}
	return false
}

func extractCommand(av dt.AtomicMap, m *dt.Msg, words,
	ssc []string) *language.WordT {
	var cmd *language.WordT
	eng := porter2.Stemmer
	for i := len(ssc) - 1; i > 0; i-- {
		// we want the command that's closest to the object, i.e. the
		// last command in the sentence
		c := ssc[i]
		if !av.Get(c) {
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

func extractObject(av dt.AtomicMap, m *dt.Msg, words, ssc []string) *language.WordT {
	var obj *language.WordT
	seen := map[string]bool{}
	for _, w := range m.Input.StructuredInput.Objects.StringSlice() {
		if !av.Get(w) {
			// this can and should be done much more efficiently
			for i, wrd := range language.RemoveStopWords(words) {
				if wrd != w {
					continue
				}
				if obj == nil {
					obj = &language.WordT{
						Index: i,
						Word:  w,
					}
				} else if !seen[w] {
					obj.Word += " " + w
				}
				seen[w] = true
			}
		}
	}
	return obj
}

func deleteRecentNodes(db *sqlx.DB, u *dt.User) error {
	if err := deleteNodes(db, u); err != nil {
		return err
	}
	q := `DELETE FROM knowledgenodes
	      WHERE userid=$1
	      ORDER BY createdat DESC
	      LIMIT 1`
	if _, err := db.Exec(q, u.ID); err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}

func deleteNodes(db *sqlx.DB, u *dt.User) error {
	q := `DELETE FROM knowledgenodes WHERE userid=$1 AND relation IS NULL`
	if _, err := db.Exec(q, u.ID); err != nil && err != sql.ErrNoRows {
		return err
	}
	return nil
}
