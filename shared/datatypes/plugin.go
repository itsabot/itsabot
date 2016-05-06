package dt

import (
	"time"

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/nlp"
	"github.com/jmoiron/sqlx"
)

// Plugin holds config options for any Abot plugin. Name must be globally
// unique. Port takes the format of ":1234". Note that the colon is
// significant. ServerAddress will default to localhost if left blank.
type Plugin struct {
	Config  PluginConfig
	Vocab   *Vocab
	Trigger *nlp.StructuredInput
	DB      *sqlx.DB
	Log     *log.Logger
	Events  *PluginEvents
	*PluginFns
}

// PluginFns defines the required functions for a plugin to be used.
type PluginFns struct {
	// Run when beginning a new conversation with a plugin. Run is a
	// required function.
	Run func(in *Msg) (string, error)

	// FollowUp runs with 2+ consecutive messages to the same plugin.
	// FollowUp is a required function.
	FollowUp func(in *Msg) (string, error)
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
}

// PluginEvents allow plugins to listen to events as they happen in Abot core.
// Simply overwrite the plugin's function
type PluginEvents struct {
	PostReceive    func(cmd *string)
	PreProcessing  func(cmd *string, u *User)
	PostProcessing func(in *Msg)
	PostResponse   func(in *Msg, resp *string)
}

// Schedule a message to the user to be delivered at a future time. This is
// particularly useful for reminders. The method of delivery, e.g. SMS or
// email, will be determined automatically by Abot at the time of sending the
// message. Abot will contact the user using that user's the most recently used
// communication method. This method returns the scheduled event ID in the
// database for future reference and an error if the event could not be
// scheduled.
func (p *Plugin) Schedule(u *User, content string, sendat time.Time) (uint64,
	error) {

	q := `INSERT INTO scheduledevents (content, userid, sendat)
	      VALUES ($1, $2, $3)
	      RETURNING id`
	var sid uint64
	err := p.DB.QueryRow(q, content, u.ID, sendat).Scan(&sid)
	return sid, err
}
