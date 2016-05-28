package core

import (
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
)

// errMissingPlugin denotes that Abot could find neither a plugin with
// matching triggers for a user's message nor any prior plugin used.
// This is most commonly seen on first run if the user's message
// doesn't initially trigger a plugin.
var errMissingPlugin = errors.New("missing plugin")

// preprocess converts a user input into a Msg that's been persisted to the
// database
func preprocess(r *http.Request) (*dt.Msg, error) {
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
func ProcessText(r *http.Request) (ret string, err error) {
	// Process message
	in, err := preprocess(r)
	if err != nil {
		return "", err
	}
	log.Debug("processed input into message...")
	log.Debug("commands:", in.StructuredInput.Commands)
	log.Debug(" objects:", in.StructuredInput.Objects)
	log.Debug(" intents:", in.StructuredInput.Intents)
	plugin, route, directRoute, followup, pluginErr := GetPlugin(db, in)
	if pluginErr != nil && pluginErr != errMissingPlugin {
		return "", pluginErr
	}
	in.Route = route
	if plugin == nil {
		in.Plugin = ""
	} else {
		in.Plugin = plugin.Config.Name
	}
	if err = in.Save(db); err != nil {
		return "", err
	}
	sendPostProcessingEvent(in)

	// Determine appropriate response
	resp := &dt.Msg{}
	resp.AbotSent = true
	resp.User = in.User
	resp.Sentence = RespondWithOffense(in)
	if len(resp.Sentence) > 0 {
		return resp.Sentence, nil
	}
	resp.Sentence = RespondWithNicety(in)
	if len(resp.Sentence) > 0 {
		if err = resp.Save(db); err != nil {
			return "", err
		}
		return resp.Sentence, nil
	}
	var smAnswered bool
	if pluginErr != errMissingPlugin {
		resp.Sentence, smAnswered = dt.CallPlugin(plugin, in, followup)
	}
	if len(resp.Sentence) == 0 {
		resp.Sentence = ConfusedLang()
		in.NeedsTraining = true
		if err = in.Update(db); err != nil {
			return "", err
		}
	} else {
		state := plugin.GetMemory(in, dt.StateKey).Int64()
		if plugin != nil && state == 0 && !directRoute && smAnswered {
			resp.Sentence = ConfusedLang()
			in.NeedsTraining = true
			if err = in.Update(db); err != nil {
				return "", err
			}
		}
	}
	if plugin != nil {
		resp.Plugin = plugin.Config.Name
	}
	sendPreResponseEvent(in, &resp.Sentence)
	if err = resp.Save(db); err != nil {
		return "", err
	}
	return resp.Sentence, nil
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

func sendPreResponseEvent(in *dt.Msg, resp *string) {
	for _, p := range AllPlugins {
		p.Events.PreResponse(in, resp)
	}
}
