package core

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/jmoiron/sqlx"
)

const (
	keyContextTime   = "__contextTime"
	keyContextPeople = "__contextPeople"

	// TODO - but first need a more accurate measure of what's an object.
	// False positives matter for this.
	// keyContextObjects = "__contextObjects"
)

// saveContext records context in the database across multiple categories.
func saveContext(db *sqlx.DB, in *dt.Msg) error {
	if err := saveTimeContext(db, in); err != nil {
		return err
	}
	if err := savePeopleContext(db, in); err != nil {
		return err
	}
	return nil
}

// saveTimeContext records contextual information about the time being
// discussed, enabling Abot to replace things like "then" with the time it
// should represent.
func saveTimeContext(db *sqlx.DB, in *dt.Msg) error {
	if len(in.StructuredInput.Times) == 0 {
		return nil
	}
	byt, err := json.Marshal(in.StructuredInput.Times)
	if err != nil {
		return err
	}
	if in.User.ID > 0 {
		q := `INSERT INTO states (key, value, userid, pluginname)
		      VALUES ($1, $2, $3, '')
		      ON CONFLICT (userid, pluginname, key)
		      DO UPDATE SET value=$2`
		_, err = db.Exec(q, keyContextTime, byt, in.User.ID)
	} else {
		q := `INSERT INTO states
		      (key, value, flexid, flexidtype, pluginname)
		      VALUES ($1, $2, $3, $4, '')
		      ON CONFLICT (flexid, flexidtype, pluginname, key)
		      DO UPDATE SET value=$2`
		_, err = db.Exec(q, keyContextTime, byt, in.User.FlexID,
			in.User.FlexIDType)
	}
	if err != nil {
		return err
	}
	return nil
}

// savePeopleContext records contextual information about people being
// discussed, enabling Abot to replace things like "him", "her", or "they" with
// the names the pronouns represent.
func savePeopleContext(db *sqlx.DB, in *dt.Msg) error {
	if len(in.StructuredInput.People) == 0 {
		return nil
	}
	byt, err := json.Marshal(in.StructuredInput.People)
	if err != nil {
		return err
	}
	if in.User.ID > 0 {
		q := `INSERT INTO states (key, value, userid, pluginname)
		      VALUES ($1, $2, $3, '')
		      ON CONFLICT (userid, pluginname, key)
		      DO UPDATE SET value=$2`
		_, err = db.Exec(q, keyContextPeople, byt, in.User.ID)
	} else {
		q := `INSERT INTO states
		      (key, value, flexid, flexidtype, pluginname)
		      VALUES ($1, $2, $3, $4, '')
		      ON CONFLICT (flexid, flexidtype, pluginname, key)
		      DO UPDATE SET value=$2`
		_, err = db.Exec(q, keyContextPeople, byt, in.User.FlexID,
			in.User.FlexIDType)
	}
	if err != nil {
		return err
	}
	return nil
}

// addContext to a Msg, filling in pronouns with the terms to which they refer.
// The sentence/stems/tokens are left unmodified; addContext simply appends the
// contextual terms to the StructuredInput when it's otherwise empty.
func addContext(db *sqlx.DB, in *dt.Msg) error {
	if len(in.StructuredInput.Times) == 0 {
		if err := addTimeContext(db, in); err != nil {
			return err
		}
	}
	if len(in.StructuredInput.People) == 0 {
		if err := addPeopleContext(db, in); err != nil {
			return err
		}
	}
	return nil
}

// addTimeContext adds a time context to a Message if the word "then" is found.
func addTimeContext(db *sqlx.DB, in *dt.Msg) error {
	var addContext bool
	for _, stem := range in.Stems {
		if stem == "then" {
			addContext = true
			break
		}
	}
	if !addContext {
		return nil
	}
	var byt []byte
	var err error
	if in.User.ID > 0 {
		q := `SELECT value FROM states WHERE userid=$1 AND key=$2`
		err = db.Get(&byt, q, in.User.ID, keyContextTime)
	} else {
		q := `SELECT value FROM states
		      WHERE flexid=$1 AND flexidtype=$2 AND key=$3`
		err = db.Get(&byt, q, in.User.FlexID, in.User.FlexIDType,
			keyContextTime)
	}
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var times []time.Time
	if err = json.Unmarshal(byt, &times); err != nil {
		return err
	}
	in.StructuredInput.Times = times
	return nil
}

// addPeopleContext adds people based on context to the sentence when
// appropriate pronouns are found, like "us", "him", "her", or "them".
func addPeopleContext(db *sqlx.DB, in *dt.Msg) error {
	var addContext, singular bool
	var sex dt.Sex
	for _, stem := range in.Stems {
		switch stem {
		case "us":
			addContext = true
		case "him", "he":
			addContext, singular = true, true
			if sex == dt.SexFemale {
				sex = dt.SexEither
			} else if sex != dt.SexEither {
				sex = dt.SexMale
			}
		case "her", "she":
			addContext, singular = true, true
			if sex == dt.SexMale {
				sex = dt.SexEither
			} else if sex != dt.SexEither {
				sex = dt.SexFemale
			}
		case "them":
			addContext = true
			sex = dt.SexEither
		}
	}
	if !addContext {
		return nil
	}
	var byt []byte
	var err error
	if in.User.ID > 0 {
		q := `SELECT value FROM states WHERE userid=$1 AND key=$2`
		err = db.Get(&byt, q, in.User.ID, keyContextPeople)
	} else {
		q := `SELECT value FROM states
		      WHERE flexid=$1 AND flexidtype=$2 AND key=$3`
		err = db.Get(&byt, q, in.User.FlexID, in.User.FlexIDType,
			keyContextPeople)
	}
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	var people []dt.Person
	if err = json.Unmarshal(byt, &people); err != nil {
		return err
	}

	// Filter our people in context by criteria, like sex.
	if !singular {
		in.StructuredInput.People = people
		return nil
	}
	if sex == dt.SexEither {
		// To reach this point, we have at least one person in context.
		in.StructuredInput.People = []dt.Person{people[0]}
		return nil
	}
	for _, person := range people {
		if person.Sex == sex {
			in.StructuredInput.People = []dt.Person{person}
			break
		}
	}
	return nil
}
