package main

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/avabot/ava/shared/datatypes"
	"github.com/dchest/stemmer/porter2"
	"github.com/jmoiron/sqlx"
)

type nodeTermT int

const (
	commandNode nodeTermT = iota + 1
	objectNode
)

type graphObj interface {
	IncrementConfidence(*sqlx.DB) error
	DecrementConfidence(*sqlx.DB) error
}

// gsMax defines the max number of edges and nodes returned from a search.
const int gsMax = 3

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
	StartNodeTerm string
	StartNodeID   uint64
	EndNodeID     uint64
	UserID        uint64
	NodePath      dt.Uint64_Slice
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
		return nil, edges
	}
	return edges, nil
}

// searchNodes runs when searchEdges fails to return at least gsMax results
func searchNodes(term string, edgesFound int) ([]node, error) {
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
