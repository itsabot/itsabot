package main

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/avabot/ava/shared/datatypes"
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

// gsMax defines the max number of edges and nodes returned from a search
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

type edge struct {
	StartNodeID uint64
	EndNodeID   uint64
	UserID      uint64
	NodePath    dt.Uint64_Slice
	Confidence  int
	CreatedAt   *time.Time
}

func searchEdges(start *node, u *dt.User) ([]edge, error) {
	var edges []edge
	q := `SELECT startnodeid, endnodeid, userid, nodepath, confidence
	      FROM knowledgeedges
	      WHERE userid=$1 AND startnodeid=$2
	      ORDER BY confidence DESC
	      LIMIT ` + strconv.FormatInt(gsMax, 10)
	if err := db.Select(&edges, q, u.ID, start.ID); err != nil {
		return nil, edges
	}
	if len(edges) >= gsMax {
		return edges, nil
	}
	q := `SELECT startnodeid, endnodeid, userid, nodepath, confidence
	      FROM knowledgeedges
	      WHERE userid<>$1 AND startnodeid=$2
	      ORDER BY confidence DESC
	      LIMIT ` + strconv.FormatInt(gsMax-len(edges), 10)
	rows, err := db.Queryx(q, u.ID, start.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tmp edge
	for rows.Next() {
		if err = rows.Scan(&tmp); err != nil {
			return nil, err
		}
		edges = append(edges, tmp)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return edges, nil
}

// searchNodes runs when searchEdges fails to return at least gsMax results
func searchNodes(term string, u *dt.User, edgesFound int) ([]node, error) {
	var nodes []node
	q := `SELECT id, term, termlength, termtype, relation, userid, createdat
	      FROM knowledgenodes
	      WHERE userid=$1 AND term=$2 AND relation IS NOT NULL
	      ORDER BY confidence
	      LIMIT ` + strconv.FormatInt(gsMax-edgesFound, 10)
	if err := db.Select(&nodes, q, u.ID, term); err != nil {
		return nil, err
	}
	if len(nodes)+edgesFound >= 3 {
		return nodes, nil
	}
	q := `SELECT id, term, termlength, termtype, relation, userid, createdat
	      FROM knowledgenodes
	      WHERE userid<>$1 AND term=$2 AND relation IS NOT NULL
	      ORDER BY confidence
	      LIMIT ` + strconv.FormatInt(gsMax-edgesFound, 10)
	rows, err := db.Queryx(q, u.ID, term, start.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tmp node
	for rows.Next() {
		if err = rows.Scan(&tmp); err != nil {
			return nil, err
		}
		nodes = append(nodes, tmp)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
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
