package core

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
)

// ErrInvalidCommand denotes that a user-inputted command could not be
// processed.
var ErrInvalidCommand = errors.New("invalid command")

// ErrMissingPlugin denotes that Abot could find neither a plugin with
// matching triggers for a user's message nor any prior plugin used.
// This is most commonly seen on first run if the user's message
// doesn't initially trigger a plugin.
var ErrMissingPlugin = errors.New("missing plugin")

// Preprocess converts a user input into a Msg that's been persisted to the
// database
func Preprocess(r *http.Request) (*dt.Msg, error) {
	req := &dt.Request{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		log.Info("could not parse empty body", err)
		return nil, err
	}
	sendPostReceiveEvent(&req.CMD)
	u, err := dt.GetUser(db, req)
	if err != nil {
		return nil, err
	}
	sendPreProcessingEvent(&req.CMD, u)
	msg := NewMsg(u, req.CMD)
	// TODO trigger training if needed (see buildInput)
	return msg, nil
}

// ProcessText is Abot's core logic. This function processes a user's message,
// routes it to the correct plugin, and handles edge cases like offensive
// language before returning a response to the user. Any user-presentable error
// is returned in the string. Errors returned from this function are not for the
// user, so they are handled by Abot explicitly on this function's return
// (logging, notifying admins, etc.).
func ProcessText(r *http.Request) (ret string, uid uint64, err error) {
	msg, err := Preprocess(r)
	if err != nil {
		return "", 0, err
	}
	log.Debug("processed input into message...")
	log.Debug("commands:", msg.StructuredInput.Commands)
	log.Debug(" objects:", msg.StructuredInput.Objects)
	log.Debug(" intents:", msg.StructuredInput.Intents)
	plugin, route, directRoute, followup, pluginErr := GetPlugin(db, msg)
	if pluginErr != nil && pluginErr != ErrMissingPlugin {
		return "", msg.User.ID, pluginErr
	}
	msg.Route = route
	if plugin == nil {
		msg.Plugin = ""
	} else {
		msg.Plugin = plugin.Config.Name
	}
	if err = msg.Save(db); err != nil {
		return "", msg.User.ID, err
	}
	sendPostProcessingEvent(msg)
	ret = RespondWithOffense(Offensive(), msg)
	if len(ret) > 0 {
		return ret, msg.User.ID, nil
	}
	if len(ret) == 0 {
		ret = RespondWithNicety(msg)
	}
	m := &dt.Msg{}
	m.AbotSent = true
	m.User = msg.User
	if len(ret) > 0 {
		m.Sentence = ret
		if err = m.Save(db); err != nil {
			return "", m.User.ID, err
		}
		return ret, msg.User.ID, nil
	}
	var smAnswered bool
	if pluginErr != ErrMissingPlugin {
		if followup {
			log.Debug("message is a followup")
		}
		ret, smAnswered = dt.CallPlugin(plugin, msg, followup)
	}
	if len(ret) == 0 {
		m.Sentence = ConfusedLang()
		msg.NeedsTraining = true
		if err = msg.Update(db); err != nil {
			return "", m.User.ID, err
		}
	} else {
		state := plugin.GetMemory(m, dt.StateKey).Int64()
		if plugin != nil && state == 0 && !directRoute && smAnswered {
			m.Sentence = ConfusedLang()
			msg.NeedsTraining = true
			if err = msg.Update(db); err != nil {
				return "", m.User.ID, err
			}
		} else {
			m.Sentence = ret
		}
	}
	if plugin != nil {
		m.Plugin = plugin.Config.Name
	}
	if err = m.Save(db); err != nil {
		return "", m.User.ID, err
	}
	sendPostResponseEvent(msg, &ret)
	return m.Sentence, m.User.ID, nil
}

func sendPostReceiveEvent(cmd *string) {
	for _, p := range AllPlugins {
		p.Events.PostReceive(cmd)
	}
}

func sendPreProcessingEvent(cmd *string, u *dt.User) {
	for _, p := range AllPlugins {
		p.Events.PreProcessing(cmd, u)
	}
}

func sendPostProcessingEvent(in *dt.Msg) {
	for _, p := range AllPlugins {
		p.Events.PostProcessing(in)
	}
}

func sendPostResponseEvent(in *dt.Msg, resp *string) {
	for _, p := range AllPlugins {
		p.Events.PostResponse(in, resp)
	}
}
