package core

import (
	"errors"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/language"
	log "github.com/itsabot/abot/shared/log"
	"github.com/labstack/echo"
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
func Preprocess(c *echo.Context) (*dt.Msg, error) {
	cmd := c.Get("cmd").(string)
	if len(cmd) == 0 {
		return nil, ErrInvalidCommand
	}
	u, err := dt.GetUser(DB(), c)
	if err != nil {
		return nil, err
	}
	msg := NewMsg(u, cmd)
	// TODO trigger training if needed (see buildInput)
	return msg, nil
}

// ProcessText is Abot's core logic. This function processes a user's message,
// routes it to the correct plugin, and handles edge cases like offensive
// language before returning a response to the user. Any user-presentable error
// is returned in the string. Errors returned from this function are not for the
// user, so they are handled by Abot explicitly on this function's return
// (logging, notifying admins, etc.).
func ProcessText(c *echo.Context) (ret string, uid uint64, err error) {
	msg, err := Preprocess(c)
	if err != nil {
		return "", 0, err
	}
	log.Debug("processed input into message...")
	log.Debug("commands:", msg.StructuredInput.Commands)
	log.Debug(" objects:", msg.StructuredInput.Objects)
	plugin, route, followup, pluginErr := GetPlugin(DB(), msg)
	if pluginErr != nil && pluginErr != ErrMissingPlugin {
		return "", msg.User.ID, pluginErr
	}
	msg.Route = route
	if plugin == nil {
		msg.Plugin = ""
	} else {
		msg.Plugin = plugin.Config.Name
	}
	if err = msg.Save(DB()); err != nil {
		return "", msg.User.ID, err
	}
	ret = RespondWithOffense(Offensive(), msg)
	if len(ret) > 0 {
		return ret, msg.User.ID, nil
	}
	if pluginErr != ErrMissingPlugin {
		if followup {
			log.Debug("message is a followup")
		}
		ret = CallPlugin(plugin, msg, followup)
	}
	responseNeeded := true
	if len(ret) == 0 {
		responseNeeded, ret = RespondWithNicety(msg)
	}
	if !responseNeeded {
		return "", msg.User.ID, nil
	}
	m := &dt.Msg{}
	m.AbotSent = true
	m.User = msg.User
	if len(ret) == 0 {
		m.Sentence = language.Confused()
		msg.NeedsTraining = true
		if err = msg.Update(DB()); err != nil {
			return "", m.User.ID, err
		}
	} else {
		m.Sentence = ret
	}
	if plugin != nil {
		m.Plugin = plugin.Config.Name
	}
	if err = m.Save(db); err != nil {
		return "", m.User.ID, err
	}
	return m.Sentence, m.User.ID, nil
}
