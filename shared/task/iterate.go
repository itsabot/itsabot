package task

import (
	"encoding/json"
	"fmt"

	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/language"
)

// OptsIterate holds the options for an iterable task.
type OptsIterate struct {
	// IterableMemKey is the key in memory where the iterable items exist.
	// The iterable memory must be a []string.
	IterableMemKey string

	// ResultMemKey is the key in memory where the selected item's index is
	// stored. Use this key to access the results of the Iterable task.
	ResultMemKeyIdx string
}

// keySelection is the currently active item index in the iterable.
const keySelection = "__iterableSelectionIdx"

// Iterate through a set of options allowing the user to select one before
// continuing.
func Iterate(p *dt.Plugin, label string, opts OptsIterate) []dt.State {
	if len(label) == 0 {
		label = "__iterableStart"
	}
	return []dt.State{
		{
			Label: label,
			OnEntry: func(in *dt.Msg) string {
				var ss []string
				mem := p.GetMemory(in, opts.IterableMemKey)
				err := json.Unmarshal(mem.Val, &ss)
				if err != nil {
					p.Log.Info("failed to get iterable memory.", err)
					return ""
				}
				if len(ss) == 0 {
					return "I'm afraid I couldn't find any results like that."
				}
				var idx int64
				if p.HasMemory(in, keySelection) {
					idx = p.GetMemory(in, keySelection).Int64() + 1
				}
				if idx >= int64(len(ss)) {
					return "I'm afraid that's all I have."
				}
				p.SetMemory(in, keySelection, idx)
				return fmt.Sprintf("How about %s?", ss[idx])
			},
			OnInput: func(in *dt.Msg) {
				yes, err := language.ExtractYesNo(in.Sentence)
				if err != nil {
					// Yes/No answer not found
					return
				}
				if yes {
					idx := p.GetMemory(in, keySelection).Int64()
					p.SetMemory(in, opts.ResultMemKeyIdx, idx)
					return
				}
				// TODO handle "next", "something else", etc.
			},
			Complete: func(in *dt.Msg) (bool, string) {
				if p.HasMemory(in, opts.ResultMemKeyIdx) {
					return true, ""
				}
				return false, p.SM.ReplayState(in)
			},
		},
	}
}

// ResetIterate should be called from within your plugin's SetOnReset function
// if you use the Iterable task.
func ResetIterate(p *dt.Plugin, in *dt.Msg) {
	p.DeleteMemory(in, keySelection)
}
