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

type MeetingState struct {
	Times  []time.Time
	Actors []Actor
	Places string
}

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
	trigger := &dt.StructuredInput{
		Commands: []string{"schedule"},
		Objects:  []string{"meeting"},
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

func (p *Meeting) Run(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	resp := m.NewResponse()
	timeString := m.Input.StructuredInput.Times.String()
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
	places := language.SliceToString(m.Input.StructuredInput.Places, "or")
	tmp := &MeetingState{
		Times:  times,
		Actors: actors,
		Places: places,
	}
	resp.State = tmp.ToMap()
	if stateIncomplete(tmp, resp) {
		return pkg.SaveResponse(respMsg, resp)
	}
	resp.Sentence = "Sure, I'll meeting that for you."
	return pkg.SaveResponse(respMsg, resp)
}

func (p *Meeting) FollowUp(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	if err := m.GetLastResponse(db); err != nil {
		return err
	}
	resp := m.LastResponse
	resp.Sentence = ""
	state := MapToMeetingState(resp.State)
	if len(state.Times) == 0 {
		timeString := m.Input.StructuredInput.Times.String()
		times, err := timeparse.Parse(timeString)
		if err != nil {
			return err
		}
		if len(times) == 0 {
			return pkg.SaveResponse(respMsg, resp)
		}
		state.Times = times
		if stateIncomplete(state, resp) {
			resp.State = state.ToMap()
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	if len(state.Actors) == 0 {
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
		if stateIncomplete(state, resp) {
			resp.State = state.ToMap()
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	if len(state.Places) == 0 {
		if len(m.Input.StructuredInput.Places) == 0 {
			resp.Sentence = ""
		} else {
			resp.State["Places"] = language.SliceToString(
				m.Input.StructuredInput.Places, "or")
		}
		if stateIncomplete(state, resp) {
			resp.State = state.ToMap()
			return pkg.SaveResponse(respMsg, resp)
		}
	}
	resp.Sentence = "Ok. I'll work with them to schedule it!"
	// TODO set a marker that it requires outside communication, follow-up
	return pkg.SaveResponse(respMsg, resp)
}

// Others handles communication with others, many of whom may not be users.
func (p *Meeting) Outside(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	return nil
}

// Repeat transactions like meeting follow-ups.
func (p *Meeting) Repeat(m *dt.Msg,
	respMsg *dt.RespMsg) error {
	return nil
}

func stateIncomplete(state *MeetingState, resp *dt.Resp) bool {
	if len(state.Times) == 0 {
		resp.Sentence = "What time would you like to have the meeting?"
		return true
	}
	if len(state.Actors) == 0 {
		resp.Sentence = "Who should I invite?"
		return true
	}
	if len(state.Places) == 0 {
		resp.Sentence = "Where do you want to have the meeting?"
		return true
	}
	return false
}

func (ms *MeetingState) ToMap() map[string]interface{} {
	mp := map[string]interface{}{}
	mp["Times"] = ms.Times
	mp["Actors"] = ms.Actors
	mp["Places"] = ms.Places
	return mp
}

func MapToMeetingState(m map[string]interface{}) *MeetingState {
	ms := MeetingState{}
	var ok bool
	ms.Times, ok = m["Times"].([]time.Time)
	if !ok {
		ms.Times = []time.Time{}
	}
	ms.Actors, ok = m["Actors"].([]Actor)
	ms.Places = m["Places"].(string)
	return &ms
}
