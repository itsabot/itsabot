package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/avabot/ava/Godeps/_workspace/src/github.com/jmoiron/sqlx"
	"github.com/avabot/ava/shared/datatypes"
	"github.com/avabot/ava/shared/helpers/timeparse"
	"github.com/avabot/ava/shared/language"
	"github.com/avabot/ava/shared/pkg"
)

var port = flag.Int("port", 0, "Port used to communicate with Ava.")
var db *sqlx.DB

type Meeting string

type Actor struct {
	Name  string
	Phone string
}

func main() {
	var err error
	db, err = pkg.ConnectDB()
	if err != nil {
		log.Fatalln("connecting to db", err)
	}
	flag.Parse()
	trigger := &datatypes.StructuredInput{
		Commands: []string{"meeting"},
	}
	p, err := pkg.NewPackage("meeting", *port, trigger)
	if err != nil {
		log.Fatalln("creating package", p.Config.Name, err)
	}
	meeting := new(Meeting)
	if err := p.Register(meeting); err != nil {
		log.Fatalln("registering package ", err)
	}
}

func (p *Meeting) Run(m *datatypes.Message,
	respMsg *datatypes.ResponseMsg) error {
	resp := m.NewResponse()
	timeString := m.Input.StructuredInput.Times.String()
	timeSlice := m.Input.StructuredInput.Times.StringSlice()
	timesString := language.SliceToString(timeSlice, "or")
	times, err := timeparse.Parse(timeString)
	if err != nil {
		return err
	}
	actors := []Actor{}
	names := []string{}
	for _, a := range m.Input.StructuredInput.Actors {
		if strings.ToLower(a) != "me" {
			actors = append(actors, Actor{Name: a})
			names = append(names, a)
		}
	}
	actorsString := language.SliceToString(names, "and")
	places := language.SliceToString(m.Input.StructuredInput.Places, "or")
	resp.State = map[string]interface{}{
		"Times":        times,
		"Actors":       actors,
		"Places":       places,
		"TimesString":  timesString,
		"ActorsString": actorsString,
	}
	if stateIncomplete(resp) {
		return pkg.SaveResponse(respMsg, resp)
	}
	resp.Sentence = "Sure, I'll meeting that for you."
	return pkg.SaveResponse(respMsg, resp)
}

func (p *Meeting) FollowUp(m *datatypes.Message,
	respMsg *datatypes.ResponseMsg) error {
	if err := m.GetLastResponse(db); err != nil {
		return err
	}
	resp := m.LastResponse
	resp.Sentence = ""
	if len(resp.State["Times"].([]time.Time)) == 0 {
		timeString := m.Input.StructuredInput.Times.String()
		timeSlice := m.Input.StructuredInput.Times.StringSlice()
		timesString := language.SliceToString(timeSlice, "or")
		times, err := timeparse.Parse(timeString)
		if err != nil {
			return err
		}
		if len(times) == 0 {
			return pkg.SaveResponse(respMsg, resp)
		}
		resp.State["Times"] = times
		resp.State["TimesString"] = timesString
		if stateIncomplete(resp) {
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	if len(resp.State["Actors"].([]Actor)) == 0 {
		actors := []Actor{}
		names := []string{}
		for _, a := range m.Input.StructuredInput.Actors {
			if strings.ToLower(a) != "me" {
				actors = append(actors, Actor{Name: a})
				names = append(names, a)
			}
		}
		actorsString := language.SliceToString(names, "and")
		resp.State["Actors"] = actors
		resp.State["ActorsString"] = actorsString
		if stateIncomplete(resp) {
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	if len(resp.State["Places"].([]string)) == 0 {
		resp.State["Places"] = m.Input.StructuredInput.Places
		if stateIncomplete(resp) {
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	return pkg.SaveResponse(respMsg, resp)
}

// Others handles communication with others, many of whom may not be users.
func (p *Meeting) Outside(m *datatypes.Message,
	respMsg *datatypes.ResponseMsg) error {
	return nil
}

// Repeat transactions like meeting follow-ups.
func (p *Meeting) Repeat(m *datatypes.Message,
	respMsg *datatypes.ResponseMsg) error {
	return nil
}

func stateIncomplete(resp *datatypes.Response) bool {
	if len(resp.State["Times"].([]time.Time)) == 0 {
		resp.Sentence = "What time would you like to have the meeting?"
		return true
	}
	if len(resp.State["Actors"].([]Actor)) == 0 {
		resp.Sentence = "Who should I invite?"
		return true
	}
	if len(resp.State["Places"].([]string)) == 0 {
		resp.Sentence = "Where do you want to have the meeting?"
		return true
	}
	return false
}
