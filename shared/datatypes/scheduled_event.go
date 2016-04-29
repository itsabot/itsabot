package dt

import (
	"fmt"

	"github.com/itsabot/abot/shared/interface/sms"
)

// ScheduledEvent for Abot to send a message at some point in the future. No
// time.Time is necessary in this struct because it's only created when it's
// known to be time to send. See core/boot:NewServer().
type ScheduledEvent struct {
	ID         uint64
	Content    string
	FlexID     string
	FlexIDType FlexIDType
}

// Send a scheduled event. Currently only phones are supported.
func (s *ScheduledEvent) Send(c *sms.Conn) error {
	switch s.FlexIDType {
	case fidtPhone:
		if err := c.Send(s.FlexID, s.Content); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized flexidtype: %d", s.FlexIDType)
	}
	return nil
}
