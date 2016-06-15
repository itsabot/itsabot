package dt

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/itsabot/abot/core/log"
	"github.com/jmoiron/sqlx"
)

// Plugin is a self-contained unit that holds everything an Abot plugin
// developer needs.
type Plugin struct {
	Config      PluginConfig
	SM          *StateMachine
	Keywords    *Keywords
	Trigger     *StructuredInput
	States      []State
	DB          *sqlx.DB
	Log         *log.Logger
	Events      *PluginEvents
	SetBranches func(in *Msg) [][]State
}

// PluginConfig holds options for a plugin.
type PluginConfig struct {
	// Name is the user-presentable name of the plugin and must be
	// unique.It's defined in plugin.json
	Name string

	// Icon is the relative path to an icon image. It's defined in
	// plugin.json.
	Icon string

	// Maintainer is the owner/publisher of the plugin.
	Maintainer string

	// ID is the remote ID of the plugin. This is pulled automatically at
	// time of plugin install from ITSABOT_URL.
	ID uint64

	// Settings is a collection of options for use by a plugin, such as a
	// required API key.
	Settings map[string]*PluginSetting
}

// PluginSetting defines whether a plugin's setting is required (empty values
// panic on boot), and whether there's a default value.
type PluginSetting struct {
	Required bool
	Default  string
}

// PluginEvents allow plugins to listen to events as they happen in Abot core.
// Simply overwrite the plugin's function
type PluginEvents struct {
	PostReceive    func(cmd *string)
	PreProcessing  func(cmd *string, u *User)
	PostProcessing func(in *Msg)
	PreResponse    func(in *Msg, resp *string)
}

// Schedule a message to the user to be delivered at a future time. This is
// particularly useful for reminders. The method of delivery, e.g. SMS or
// email, will be determined automatically by Abot at the time of sending the
// message. Abot will contact the user using that user's the most recently used
// communication method. This method returns an error if the event could not be
// scheduled.
func (p *Plugin) Schedule(in *Msg, content string, sendat time.Time) error {
	if sendat.Before(time.Now()) {
		return errors.New("cannot schedule time in the past")
	}
	q := `INSERT INTO scheduledevents (content, flexid, flexidtype, sendat,
		pluginname)
	      VALUES ($1, $2, $3, $4, $5)`
	_, err := p.DB.Exec(q, content, in.User.FlexID, in.User.FlexIDType, sendat, p.Config.Name)
	return err
}

// run is an unexported function that executes a plugin's behavior when the
// plugin is called. First the plugin attempts to respond using keyword
// functions. If that response is empty (""), run will try the plugin's state
// machine. If that response is also empty, the plugin passes that empty
// response to Abot, which often results in Abot responding to the user with
// confusion. The returned bool notifies the caller whether the response is from
// the state machine.
func (p *Plugin) run(in *Msg) (resp string, stateMachineAnswered bool) {
	resp = p.Keywords.handle(in)
	if len(resp) == 0 {
		// Copy the plugin's state machine by value into this local
		// variable to enable different users to have different
		// branching conversations at the same time (that's all kept in
		// local--and user-specific--memory here).
		sm := &StateMachine{}
		*sm = *p.SM
		states := p.SetBranches(in)
		if states != nil {
			sm.SetStates(states)
		}
		sm.LoadState(in)
		if sm.state < len(sm.Handlers) {
			resp = sm.Next(in)
			stateMachineAnswered = true
		}
	}
	return resp, stateMachineAnswered
}

// CallPlugin sends a plugin the user's preprocessed message. The followup bool
// dictates whether this is the first consecutive time the user has sent that
// plugin a message, or if the user is engaged in a conversation with the
// plugin. This difference enables plugins to respond differently--like reset
// state--when messaged for the first time in each new conversation.
func CallPlugin(p *Plugin, in *Msg, followup bool) (resp string,
	stateMachineAnswered bool) {

	if p == nil {
		return "", false
	}
	if !followup {
		p.SM.Reset(in)
	}
	return p.run(in)
}

// GetMemory retrieves a memory for a given key. Accessing that Memory's value
// is described in itsabot.org/abot/shared/datatypes/memory.go.
func (p *Plugin) GetMemory(in *Msg, k string) Memory {
	var buf []byte
	var err error

	// Only retrieve state-related values of that specific plugin.
	// Otherwise fetch the memory from any plugin.
	if in.User.ID > 0 {
		if k == StateKey || k == stateEnteredKey {
			q := `SELECT value FROM states
			      WHERE userid=$1 AND key=$2 AND pluginname=$3`
			err = p.DB.Get(&buf, q, in.User.ID, k, p.Config.Name)
		} else {
			q := `SELECT value FROM states
			      WHERE userid=$1 AND key=$2`
			err = p.DB.Get(&buf, q, in.User.ID, k)
		}
	} else {
		if k == StateKey || k == stateEnteredKey {
			q := `SELECT value FROM states
			      WHERE flexid=$1 AND flexidtype=$2 AND key=$3 AND pluginname=$4`
			err = p.DB.Get(&buf, q, in.User.FlexID,
				in.User.FlexIDType, k, p.Config.Name)
		} else {
			q := `SELECT value FROM states
			      WHERE flexid=$1 AND flexidtype=$2 AND key=$3`
			err = p.DB.Get(&buf, q, in.User.FlexID,
				in.User.FlexIDType, k)
		}
	}
	if err == sql.ErrNoRows {
		return Memory{Key: k, Val: json.RawMessage{}, log: p.Log}
	}
	if err != nil {
		p.Log.Infof("could not get memory for key %s. %s", k,
			err.Error())
		return Memory{Key: k, Val: json.RawMessage{}, log: p.Log}
	}
	return Memory{Key: k, Val: buf, log: p.Log}
}

// SetMemory saves to some key to some value in Abot's memory, which can be
// accessed by any state or plugin. Memories are stored in a key-value format,
// and any marshalable/unmarshalable datatype can be stored and retrieved.
// Note that Ava's memory is global, peristed across plugins. This enables
// plugins that subscribe to an agreed-upon memory API to communicate between
// themselves. Thus, if it's absolutely necessary that no some other plugins
// modify or access a memory, use a long key unlikely to collide with any other
// plugin's.
func (p *Plugin) SetMemory(in *Msg, k string, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		p.Log.Infof("could not marshal memory interface to json at %s. %s",
			k, err.Error())
		return
	}
	p.Log.Debug("setting memory for", k, "to", string(b))
	if in.User.ID > 0 {
		q := `INSERT INTO states (key, value, pluginname, userid)
		      VALUES ($1, $2, $3, $4)
		      ON CONFLICT (userid, pluginname, key)
		      DO UPDATE SET value=$2`
		_, err = p.DB.Exec(q, k, b, p.Config.Name, in.User.ID)
	} else {
		q := `INSERT INTO states
		      (key, value, pluginname, flexid, flexidtype)
		      VALUES ($1, $2, $3, $4, $5)
		      ON CONFLICT (flexid, flexidtype, pluginname, key)
		      DO UPDATE SET value=$2`
		_, err = p.DB.Exec(q, k, b, p.Config.Name, in.User.FlexID,
			in.User.FlexIDType)
	}
	if err != nil {
		p.Log.Infof("could not set memory at %s to %s. %s", k, v,
			err.Error())
		return
	}
}

// DeleteMemory deletes a memory for a given key. It is not an error to delete
// a key that does not exist.
func (p *Plugin) DeleteMemory(in *Msg, k string) {
	var err error
	if in.User.ID > 0 {
		q := `DELETE FROM states
		      WHERE userid=$1 AND pluginname=$2 AND key=$3`
		_, err = p.DB.Exec(q, in.User.ID, p.Config.Name, k)
	} else {
		q := `DELETE FROM states
		      WHERE flexid=$1 AND flexidtype=$2 AND pluginname=$3
		      AND key=$4`
		_, err = p.DB.Exec(q, in.User.FlexID, in.User.FlexIDType,
			p.Config.Name, k)
	}
	if err != nil {
		p.Log.Infof("could not delete memory for key %s. %s", k,
			err.Error())
	}
}

// GetSetting retrieves a specific setting's value. It throws a fatal error if
// the setting has not been declared in the plugin's plugin.json file.
func (p *Plugin) GetSetting(name string) string {
	if p.Config.Settings[name] == nil {
		m := fmt.Sprintf(
			"missing setting %s. please declare it in the %s's plugin.json",
			name, p.Config.Name)
		log.Fatal(m)
	}
	var val string
	q := `SELECT value FROM settings WHERE name=$1 AND pluginname=$2`
	err := p.DB.Get(&val, q, name, p.Config.Name)
	if err == sql.ErrNoRows {
		return p.Config.Settings[name].Default
	}
	if err != nil {
		log.Info("failed to get plugin setting.", err)
		return ""
	}
	return val
}

// HasMemory is a helper function to simply a common use-case, determing if some
// key/value has been set in Ava, i.e. if the memory exists.
func (p *Plugin) HasMemory(in *Msg, k string) bool {
	return len(p.GetMemory(in, k).Val) > 0
}
