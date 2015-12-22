package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/dchest/stemmer/porter2"
	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
)

type nodeTermT int

const (
	nodeTermCommand nodeTermT = iota + 1
	nodeTermObject
)

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

func searchEdgesForTerm(term string) ([]edge, error) {
	var edges []edge
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

func searchEdges(start *node) ([]edge, error) {
	var edges []edge
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
func searchNodes(term string, edgesFound int64) ([]node, error) {
	var nodes []node
	q := `SELECT id, term, termlength, termtype, relation, userid, createdat
	      FROM knowledgenodes
	      WHERE term=$1 AND relation IS NOT NULL
	      ORDER BY confidence
	      LIMIT ` + strconv.FormatInt(gsMax-edgesFound, 10)
	if err := db.Select(&nodes, q, term); err != nil {
		return nil, err
	}
	return nodes, nil
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
	return n.Relation.String
}

func (n *node) Text() string {
	var s string
	if n.TermType == nodeTermObject {
		s = "What's " + n.Term + "?"
	} else if qry.WordType == nodeTermCommand {
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
		if len(ss) > i+1 {
			break
		}
		ss = append(ss, ss[i]+" "+ss[i+1])
	}
	return ss
}

func replaceSentence(in *dt.Input, g graphObj) string {
	tmp := []string{}
	for _, w := range in.SentenceFields {
		if w == g.Trm() {
			tmp = append(tmp, g.Rel())
		}
		tmp = append(tmp, w)
	}
	return updateSentence(tmp)
}

func updateSentence(sentenceFields []string) string {
	if len(sentenceFields) == 0 {
		return ""
	}
	// almost certainly could be done in 1 loop instead of 2, avoiding the
	// separate strings.Join
	var tmp string
	for i := 1; i < len(sentenceFields); i++ {
		tmp = sentenceFields[i]
		switch tmp {
		case "\"", "'", ",", ".", ":", ";", "!", "?":
			sentenceFields[i-1] = sentenceFields[i-1] + tmp
			sentenceFields[i] = ""
		}
	}
	return strings.Join(sentenceFields, " ")
}

func getActiveNode(db *sqlx.DB, u *dt.User) (*node, error) {
	n := &node{}
	q := `SELECT id, userid, term, termtype
	      FROM knowledgenodes
	      WHERE userid=$1 AND relation IS NULL`
	err := db.Get(n, q, u.ID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return n, err
}

func newNodes(db *sqlx.DB, p *pkg.Pkg, m *dt.Msg) ([]node, error) {
	log.Debugln("creating new knowledge nodes")
	words := strings.Fields(m.Input.Sentence)
	ssc := m.Input.StructuredInput.Commands.StringSlice()
	sso := m.Input.StructuredInput.Objects.StringSlice()
	log.Debugln("sso", sso)
	cmd := extractCommand(p, m, words, ssc)
	obj := extractObject(p, m, words, sso)
	var nodes []nodes
	q := `INSERT INTO knowledgenodes
	      (term, termlength, termtype, userid) VALUES ($1, $2, $3, $4)
	      RETURNING id`
	var id uint64
	if cmd != nil {
		log.Debugln("building knowledge node around command")
		err := db.QueryRowx(q, cmd, len(cmd), nodeTermCommand,
			m.User.ID).Scan(&id)
		if err != nil {
			return nodes, err
		}
		n := node{
			ID: id,
			Term: cmd,
			TermLength: len(cmd),
			TermType: nodeTermCommand,
		}
		nodes = append(nodes, n)
	}
	if obj != nil {
		log.Debugln("building knowledge node around object")
		log.Debugln("obj", obj.Word)
		err := db.QueryRowx(q, obj, len(obj), nodeTermObject,
			m.User.ID).Scan(&id)
		if err != nil {
			return nodes, err
		}
		n := node{
			ID: id,
			Term: obj,
			TermLength: len(obj),
			TermType: nodeTermObject,
		}
		nodes = append(nodes, n)
	}
	log.Debugln("built", len(nodes), "knowledge nodes")
	if len(nodes) == 0 {
		return nodes, errors.New("no new nodes")
	}
	return nodes, nil
}

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
