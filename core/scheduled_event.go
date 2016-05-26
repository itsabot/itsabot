package core

import (
	"time"

	"github.com/itsabot/abot/core/log"
	"github.com/itsabot/abot/shared/datatypes"
)

// sendEvents recursively calls itself to continue running.
func sendEvents(evtChan chan *dt.ScheduledEvent, interval time.Duration) {
	t := time.NewTicker(time.Minute)
	select {
	case now := <-t.C:
		sendEventsTick(evtChan, now)
		sendEvents(evtChan, interval)
	}
}

func sendEventsTick(evtChan chan *dt.ScheduledEvent, t time.Time) {
	// Listen for events that need to be sent.
	go func(chan *dt.ScheduledEvent) {
		q := `UPDATE scheduledevents SET sent=TRUE WHERE id=$1`
		select {
		case evt := <-evtChan:
			log.Debug("received event")
			if smsConn == nil {
				log.Info("failed to send scheduled event (missing SMS driver). will retry.")
				return
			}
			// Send event. On error, event will be retried next
			// minute.
			if err := evt.Send(smsConn); err != nil {
				log.Info("failed to send scheduled event", err)
				return
			}
			// Update event as sent
			if _, err := db.Exec(q, evt.ID); err != nil {
				log.Info("failed to update scheduled event as sent",
					err)
				return
			}
		}
	}(evtChan)

	q := `SELECT id, content, flexid, flexidtype
		      FROM scheduledevents
		      WHERE sent=false AND sendat<=$1`
	evts := []*dt.ScheduledEvent{}
	if err := db.Select(&evts, q, time.Now()); err != nil {
		log.Info("failed to queue scheduled event", err)
		return
	}
	for _, evt := range evts {
		// Queue the event for sending
		evtChan <- evt
	}
}
