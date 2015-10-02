package main

import "github.com/avabot/ava/shared/datatypes"

func saveStructuredInput(si *datatypes.StructuredInput, rsp, pkg string) error {
	q := `
		INSERT INTO inputs (
			userid,
			flexid,
			flexidtype,
			commands,
			objects,
			actors,
			times,
			places,
			response,
			package
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	c := array(si.Commands)
	o := array(si.Objects)
	a := array(si.Actors)
	t := array(si.Times)
	p := array(si.Places)
	_, err := db.Exec(
		q, si.UserId, si.FlexId, si.FlexIdType, c, o, a, t, p, rsp, pkg)
	return err
}

func array(ss []string) string {
	if len(ss) == 0 {
		return "{}"
	}
	s := "{"
	for _, w := range ss {
		s += `"` + w + `"` + ","
	}
	return s[:len(s)-1] + "}"
}
