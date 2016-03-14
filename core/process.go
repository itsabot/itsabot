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
	plugin, route, followup, err := GetPlugin(DB(), msg)
	if err != nil {
		return "", msg.User.ID, err
	}
	msg.Route = route
	if plugin == nil {
		msg.Plugin = ""
	} else {
		msg.Plugin = plugin.P.Config.Name
	}
	if err = msg.Save(DB()); err != nil {
		return "", msg.User.ID, err
	}
	ret = RespondWithOffense(Offensive(), msg)
	if len(ret) == 0 {
		if followup {
			log.Debug("message is a followup")
		}
		ret, err = CallPlugin(plugin, msg, followup)
		if err != nil {
			return "", msg.User.ID, err
		}
		responseNeeded := true
		if len(ret) == 0 {
			responseNeeded, ret = RespondWithNicety(msg)
		}
		if !responseNeeded {
			return "", msg.User.ID, nil
		}
	}
	log.Debug("message response:", ret)
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
		m.Plugin = plugin.P.Config.Name
	}
	if err = m.Save(db); err != nil {
		return "", m.User.ID, err
	}
	return m.Sentence, m.User.ID, nil
}
