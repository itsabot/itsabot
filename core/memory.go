package core

import (
	"errors"

	"github.com/itsabot/abot/shared/datatypes"
)

// getMemory retrieves memories independent of any plugin or input to be used
// internally by Abot.
func getMemory(uid uint64, fid string, fidT dt.FlexIDType) error {
	return errors.New("getMemory not implemented")
}
