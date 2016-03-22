package dt

import (
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

	// Type specifies the type of plugin and can be either "action" or
	// "driver". It's defined in plugin.json.
	Type string
}

// PluginEvents allow plugins to listen to events as they happen in Abot core.
// Simply overwrite the plugin's function
type PluginEvents struct {
	PostReceive    func(cmd *string)
	PostProcessing func(in *Msg)
	PostResponse   func(in *Msg, resp *string)
}
