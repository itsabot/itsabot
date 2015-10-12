// go implementation of upr client.
// See https://github.com/couchbaselabs/cbupr/blob/master/transport-spec.md
// TODO
// 1. Use a pool allocator to avoid garbage
package memcached

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/couchbase/gomemcached"
	"log"
	"strconv"
	"sync"
)

const uprMutationExtraLen = 30
const uprDeletetionExtraLen = 18
const uprSnapshotExtraLen = 20
const bufferAckThreshold = 0.2
const opaqueOpen = 0xBEAF0001
const opaqueFailover = 0xDEADBEEF

// UprEvent memcached events for UPR streams.
type UprEvent struct {
	Opcode       gomemcached.CommandCode // Type of event
	Status       gomemcached.Status      // Response status
	VBucket      uint16                  // VBucket this event applies to
	Opaque       uint16                  // 16 MSB of opaque
	VBuuid       uint64                  // This field is set by downstream
	Flags        uint32                  // Item flags
	Expiry       uint32                  // Item expiration time
	Key, Value   []byte                  // Item key/value
	OldValue     []byte                  // TODO: TBD: old document value
	Cas          uint64                  // CAS value of the item
	Seqno        uint64                  // sequence number of the mutation
	RevSeqno     uint64                  // rev sequence number : deletions
	LockTime     uint32                  // Lock time
	MetadataSize uint16                  // Metadata size
	SnapstartSeq uint64                  // start sequence number of this snapshot
	SnapendSeq   uint64                  // End sequence number of the snapshot
	SnapshotType uint32                  // 0: disk 1: memory
	FailoverLog  *FailoverLog            // Failover log containing vvuid and sequnce number
	Error        error                   // Error value in case of a failure
}

// UprStream is per stream data structure over an UPR Connection.
type UprStream struct {
	Vbucket   uint16 // Vbucket id
	Vbuuid    uint64 // vbucket uuid
	StartSeq  uint64 // start sequence number
	EndSeq    uint64 // end sequence number
	connected bool
}

// UprFeed represents an UPR feed. A feed contains a connection to a single
// host and multiple vBuckets
type UprFeed struct {
	mu          sync.RWMutex
	C           <-chan *UprEvent            // Exported channel for receiving UPR events
	vbstreams   map[uint16]*UprStream       // vb->stream mapping
	closer      chan bool                   // closer
	conn        *Client                     // connection to UPR producer
	Error       error                       // error
	bytesRead   uint64                      // total bytes read on this connection
	toAckBytes  uint32                      // bytes client has read
	maxAckBytes uint32                      // Max buffer control ack bytes
	stats       UprStats                    // Stats for upr client
	transmitCh  chan *gomemcached.MCRequest // transmit command channel
	transmitCl  chan bool                   //  closer channel for transmit go-routine
}

type UprStats struct {
	TotalBytes         uint64
	TotalMutation      uint64
	TotalBufferAckSent uint64
	TotalSnapShot      uint64
}

// FailoverLog containing vvuid and sequnce number
type FailoverLog [][2]uint64

// error codes
var ErrorInvalidLog = errors.New("couchbase.errorInvalidLog")

func (flogp *FailoverLog) Latest() (vbuuid, seqno uint64, err error) {
	if flogp != nil {
		flog := *flogp
		latest := flog[len(flog)-1]
		return latest[0], latest[1], nil
	}
	return vbuuid, seqno, ErrorInvalidLog
}

func makeUprEvent(rq gomemcached.MCRequest, stream *UprStream) *UprEvent {
	event := &UprEvent{
		Opcode:  rq.Opcode,
		VBucket: stream.Vbucket,
		VBuuid:  stream.Vbuuid,
		Key:     rq.Key,
		Value:   rq.Body,
		Cas:     rq.Cas,
	}
	// 16 LSBits are used by client library to encode vbucket number.
	// 16 MSBits are left for application to multiplex on opaque value.
	event.Opaque = appOpaque(rq.Opaque)

	if len(rq.Extras) >= uprMutationExtraLen &&
		event.Opcode == gomemcached.UPR_MUTATION {

		event.Seqno = binary.BigEndian.Uint64(rq.Extras[:8])
		event.RevSeqno = binary.BigEndian.Uint64(rq.Extras[8:16])
		event.Flags = binary.BigEndian.Uint32(rq.Extras[16:20])
		event.Expiry = binary.BigEndian.Uint32(rq.Extras[20:24])
		event.LockTime = binary.BigEndian.Uint32(rq.Extras[24:28])
		event.MetadataSize = binary.BigEndian.Uint16(rq.Extras[28:30])

	} else if len(rq.Extras) >= uprDeletetionExtraLen &&
		event.Opcode == gomemcached.UPR_DELETION ||
		event.Opcode == gomemcached.UPR_EXPIRATION {

		event.Seqno = binary.BigEndian.Uint64(rq.Extras[:8])
		event.RevSeqno = binary.BigEndian.Uint64(rq.Extras[8:16])
		event.MetadataSize = binary.BigEndian.Uint16(rq.Extras[16:18])

	} else if len(rq.Extras) >= uprSnapshotExtraLen &&
		event.Opcode == gomemcached.UPR_SNAPSHOT {

		event.SnapstartSeq = binary.BigEndian.Uint64(rq.Extras[:8])
		event.SnapendSeq = binary.BigEndian.Uint64(rq.Extras[8:16])
		event.SnapshotType = binary.BigEndian.Uint32(rq.Extras[16:20])
	}

	return event
}

func (event *UprEvent) String() string {
	name := gomemcached.CommandNames[event.Opcode]
	if name == "" {
		name = fmt.Sprintf("#%d", event.Opcode)
	}
	return name
}

func sendCommands(mc *Client, ch chan *gomemcached.MCRequest, closer chan bool) {
loop:
	for {
		select {
		case command := <-ch:
			if err := mc.Transmit(command); err != nil {
				log.Printf("Failed to transmit command %s. Error %s", command.Opcode.String(), err.Error())
				break loop
			}

		case <-closer:
			break loop
		}
	}
}

// NewUprFeed creates a new UPR Feed.
// TODO: Describe side-effects on bucket instance and its connection pool.
func (mc *Client) NewUprFeed() (*UprFeed, error) {

	feed := &UprFeed{
		conn:       mc,
		closer:     make(chan bool),
		vbstreams:  make(map[uint16]*UprStream),
		transmitCh: make(chan *gomemcached.MCRequest),
		transmitCl: make(chan bool),
	}

	go sendCommands(mc, feed.transmitCh, feed.transmitCl)
	return feed, nil
}

func doUprOpen(mc *Client, name string, sequence uint32) error {

	rq := &gomemcached.MCRequest{
		Opcode: gomemcached.UPR_OPEN,
		Key:    []byte(name),
		Opaque: opaqueOpen,
	}

	rq.Extras = make([]byte, 8)
	binary.BigEndian.PutUint32(rq.Extras[:4], sequence)

	// flags = 0 for consumer
	binary.BigEndian.PutUint32(rq.Extras[4:], 1)

	if err := mc.Transmit(rq); err != nil {
		return err
	}

	if res, err := mc.Receive(); err != nil {
		return err
	} else if res.Opcode != gomemcached.UPR_OPEN {
		return fmt.Errorf("unexpected #opcode %v", res.Opcode)
	} else if rq.Opaque != res.Opaque {
		return fmt.Errorf("opaque mismatch, %v over %v", res.Opaque, res.Opaque)
	} else if res.Status != gomemcached.SUCCESS {
		return fmt.Errorf("error %v", res.Status)
	}

	return nil
}

// UprOpen to connect with a UPR producer.
// Name: name of te UPR connection
// sequence: sequence number for the connection
// bufsize: max size of the application
func (feed *UprFeed) UprOpen(name string, sequence uint32, bufSize uint32) error {
	mc := feed.conn

	if err := doUprOpen(mc, name, sequence); err != nil {
		return err
	}

	// send a UPR control message to set the window size for the this connection
	if bufSize > 0 {
		rq := &gomemcached.MCRequest{
			Opcode: gomemcached.UPR_CONTROL,
			Key:    []byte("connection_buffer_size"),
			Body:   []byte(strconv.Itoa(int(bufSize))),
		}
		feed.transmitCh <- rq
		feed.maxAckBytes = uint32(bufferAckThreshold * float32(bufSize))
	}

	return nil
}

// UprGetFailoverLog for given list of vbuckets.
func (mc *Client) UprGetFailoverLog(
	vb []uint16) (map[uint16]*FailoverLog, error) {

	rq := &gomemcached.MCRequest{
		Opcode: gomemcached.UPR_FAILOVERLOG,
		Opaque: opaqueFailover,
	}

	if err := doUprOpen(mc, "FailoverLog", 0); err != nil {
		return nil, fmt.Errorf("UPR_OPEN Failed %s", err.Error())
	}

	failoverLogs := make(map[uint16]*FailoverLog)
	for _, vBucket := range vb {
		rq.VBucket = vBucket
		if err := mc.Transmit(rq); err != nil {
			return nil, err
		}
		res, err := mc.Receive()

		if err != nil {
			return nil, fmt.Errorf("failed to receive %s", err.Error())
		} else if res.Opcode != gomemcached.UPR_FAILOVERLOG || res.Status != gomemcached.SUCCESS {
			return nil, fmt.Errorf("unexpected #opcode %v", res.Opcode)
		}

		flog, err := parseFailoverLog(res.Body)
		if err != nil {
			return nil, fmt.Errorf("unable to parse failover logs for vb %d", vb)
		}
		failoverLogs[vBucket] = flog
	}

	return failoverLogs, nil
}

// UprRequestStream for a single vbucket.
func (feed *UprFeed) UprRequestStream(vbno, opaqueMSB uint16, flags uint32,
	vuuid, startSequence, endSequence, snapStart, snapEnd uint64) error {

	rq := &gomemcached.MCRequest{
		Opcode:  gomemcached.UPR_STREAMREQ,
		VBucket: vbno,
		Opaque:  composeOpaque(vbno, opaqueMSB),
	}

	rq.Extras = make([]byte, 48) // #Extras
	binary.BigEndian.PutUint32(rq.Extras[:4], flags)
	binary.BigEndian.PutUint32(rq.Extras[4:8], uint32(0))
	binary.BigEndian.PutUint64(rq.Extras[8:16], startSequence)
	binary.BigEndian.PutUint64(rq.Extras[16:24], endSequence)
	binary.BigEndian.PutUint64(rq.Extras[24:32], vuuid)
	binary.BigEndian.PutUint64(rq.Extras[32:40], snapStart)
	binary.BigEndian.PutUint64(rq.Extras[40:48], snapEnd)

	feed.mu.Lock()
	defer feed.mu.Unlock()

	if err := feed.conn.Transmit(rq); err != nil {
		log.Printf("Error in StreamRequest %s", err.Error())
		return err
	}

	stream := &UprStream{
		Vbucket:  vbno,
		Vbuuid:   vuuid,
		StartSeq: startSequence,
		EndSeq:   endSequence,
	}
	feed.vbstreams[vbno] = stream
	return nil
}

// CloseStream for specified vbucket.
func (feed *UprFeed) CloseStream(vbno, opaqueMSB uint16) error {
	feed.mu.Lock()
	defer feed.mu.Unlock()

	if feed.vbstreams[vbno] == nil {
		return fmt.Errorf("Stream for vb %d has not been requested", vbno)
	}
	closeStream := &gomemcached.MCRequest{
		Opcode:  gomemcached.UPR_CLOSESTREAM,
		VBucket: vbno,
		Opaque:  composeOpaque(vbno, opaqueMSB),
	}
	feed.transmitCh <- closeStream
	return nil
}

// StartFeed to start the upper feed.
func (feed *UprFeed) StartFeed() error {
	return feed.StartFeedWithConfig(10)
}

func (feed *UprFeed) StartFeedWithConfig(datachan_len int) error {
	ch := make(chan *UprEvent, datachan_len)
	feed.C = ch
	go feed.runFeed(ch)
	return nil
}

func parseFailoverLog(body []byte) (*FailoverLog, error) {

	if len(body)%16 != 0 {
		err := fmt.Errorf("invalid body length %v, in failover-log", len(body))
		return nil, err
	}
	log := make(FailoverLog, len(body)/16)
	for i, j := 0, 0; i < len(body); i += 16 {
		vuuid := binary.BigEndian.Uint64(body[i : i+8])
		seqno := binary.BigEndian.Uint64(body[i+8 : i+16])
		log[j] = [2]uint64{vuuid, seqno}
		j++
	}
	return &log, nil
}

func handleStreamRequest(
	res *gomemcached.MCResponse,
) (gomemcached.Status, uint64, *FailoverLog, error) {

	var rollback uint64
	var err error

	switch {
	case res.Status == gomemcached.ROLLBACK:
		log.Printf("Rollback response. body=%v\n",res.Body)
		
		rollback = binary.BigEndian.Uint64(res.Body)
		log.Printf("Rollback %v for vb %v\n", rollback, res.Opaque)
		return res.Status, rollback, nil, nil

	case res.Status != gomemcached.SUCCESS:
		err = fmt.Errorf("unexpected status %v, for %v", res.Status, res.Opaque)
		return res.Status, 0, nil, err
	}

	flog, err := parseFailoverLog(res.Body[:])
	return res.Status, rollback, flog, err
}

// generate stream end responses for all active vb streams
func (feed *UprFeed) doStreamClose(ch chan *UprEvent) {
	feed.mu.RLock()
	for vb, stream := range feed.vbstreams {
		uprEvent := &UprEvent{
			VBucket: vb,
			VBuuid:  stream.Vbuuid,
			Opcode:  gomemcached.UPR_STREAMEND,
		}
		ch <- uprEvent
	}
	feed.mu.RUnlock()
}

func (feed *UprFeed) runFeed(ch chan *UprEvent) {
	defer close(ch)
	var headerBuf [gomemcached.HDR_LEN]byte
	var pkt gomemcached.MCRequest
	var event *UprEvent

	mc := feed.conn.Hijack()
	uprStats := &feed.stats

loop:
	for {
		sendAck := false
		bytes, err := pkt.Receive(mc, headerBuf[:])
		if err != nil {
			log.Printf("Error in receive %s", err.Error())
			feed.Error = err
			// send all the stream close messages to the client
			feed.doStreamClose(ch)
			break loop
		} else {
			event = nil
			res := &gomemcached.MCResponse{
				Opcode: pkt.Opcode,
				Cas:    pkt.Cas,
				Opaque: pkt.Opaque,
				Status: gomemcached.Status(pkt.VBucket),
				Extras: pkt.Extras,
				Key:    pkt.Key,
				Body:   pkt.Body,
			}

			vb := vbOpaque(pkt.Opaque)
			uprStats.TotalBytes = uint64(bytes)

			feed.mu.RLock()
			stream := feed.vbstreams[vb]
			feed.mu.RUnlock()

			switch pkt.Opcode {
			case gomemcached.UPR_STREAMREQ:
				if stream == nil {
					log.Printf("Stream not found for vb %d: %#v", vb, pkt)
					break loop
				}
				status, rb, flog, err := handleStreamRequest(res)
				if status == gomemcached.ROLLBACK {
					event = makeUprEvent(pkt, stream)
					event.Status = status
					
					// rollback stream
					log.Printf("UPR_STREAMREQ with rollback %d for vb %d Failed: %v", rb, vb, err)
					// delete the stream from the vbmap for the feed
					feed.mu.Lock()
					delete(feed.vbstreams, vb)
					feed.mu.Unlock()

				} else if status == gomemcached.SUCCESS {
					event = makeUprEvent(pkt, stream)
					event.Seqno = stream.StartSeq
					event.FailoverLog = flog
					event.Status = status
					
					stream.connected = true
					log.Printf("UPR_STREAMREQ for vb %d successful", vb)

				} else if err != nil {
					log.Printf("UPR_STREAMREQ for vbucket %d erro %s", vb, err.Error())
					event = &UprEvent{
						Opcode:  gomemcached.UPR_STREAMREQ,
						Status:  status,
						VBucket: vb,
						Error:   err,
					}
					// delete the stream
					feed.mu.Lock()
					delete(feed.vbstreams, vb)
					feed.mu.Unlock()
				}

			case gomemcached.UPR_MUTATION,
				gomemcached.UPR_DELETION,
				gomemcached.UPR_EXPIRATION:
				if stream == nil {
					log.Printf("Stream not found for vb %d: %#v", vb, pkt)
					break loop
				}
				event = makeUprEvent(pkt, stream)
				uprStats.TotalMutation++
				sendAck = true

			case gomemcached.UPR_STREAMEND:
				if stream == nil {
					log.Printf("Stream not found for vb %d: %#v", vb, pkt)
					break loop
				}
				//stream has ended
				event = makeUprEvent(pkt, stream)
				log.Printf("Stream Ended for vb %d", vb)
				sendAck = true

				feed.mu.Lock()
				delete(feed.vbstreams, vb)
				feed.mu.Unlock()

			case gomemcached.UPR_SNAPSHOT:
				if stream == nil {
					log.Printf("Stream not found for vb %d: %#v", vb, pkt)
					break loop
				}
				// snapshot marker
				event = makeUprEvent(pkt, stream)
				uprStats.TotalSnapShot++
				sendAck = true

			case gomemcached.UPR_FLUSH:
				if stream == nil {
					log.Printf("Stream not found for vb %d: %#v", vb, pkt)
					break loop
				}
				// special processing for flush ?
				event = makeUprEvent(pkt, stream)

			case gomemcached.UPR_CLOSESTREAM:
				if stream == nil {
					log.Printf("Stream not found for vb %d: %#v", vb, pkt)
					break loop
				}
				event = makeUprEvent(pkt, stream)
				event.Opcode = gomemcached.UPR_STREAMEND // opcode re-write !!
				log.Printf("Stream Closed for vb %d StreamEnd simulated", vb)
				sendAck = true

				feed.mu.Lock()
				delete(feed.vbstreams, vb)
				feed.mu.Unlock()

			case gomemcached.UPR_ADDSTREAM:
				log.Printf("Opcode %v not implemented", pkt.Opcode)

			case gomemcached.UPR_CONTROL, gomemcached.UPR_BUFFERACK:
				if res.Status != gomemcached.SUCCESS {
					log.Printf("Opcode %v received status %d", pkt.Opcode.String(), res.Status)
				}

			case gomemcached.UPR_NOOP:
				// send a NOOP back
				noop := &gomemcached.MCRequest{
					Opcode: gomemcached.UPR_NOOP,
				}
				feed.transmitCh <- noop

			default:
				log.Printf("Recived an unknown response for vbucket %d", vb)
			}
		}

		if event != nil {
			select {
			case ch <- event:
			case <-feed.closer:
				break loop
			}

			feed.mu.RLock()
			l := len(feed.vbstreams)
			feed.mu.RUnlock()

			if event.Opcode == gomemcached.UPR_CLOSESTREAM && l == 0 {
				log.Printf("No more streams")
			}
		}

		needToSend, sendSize := feed.SendBufferAck(sendAck, uint32(bytes))
		if needToSend {
			bufferAck := &gomemcached.MCRequest{
				Opcode: gomemcached.UPR_BUFFERACK,
			}
			bufferAck.Extras = make([]byte, 4)
			binary.BigEndian.PutUint32(bufferAck.Extras[:4], uint32(sendSize))
			feed.transmitCh <- bufferAck
			uprStats.TotalBufferAckSent++
		}
	}

	feed.transmitCl <- true
}

// Send buffer ack
func (feed *UprFeed) SendBufferAck(sendAck bool, bytes uint32) (bool, uint32) {
	if sendAck {
		totalBytes := feed.toAckBytes + bytes
		if totalBytes > feed.maxAckBytes {
			feed.toAckBytes = 0
			return true, totalBytes
		}
		feed.toAckBytes += bytes
	}
	return false, 0
}

func (feed *UprFeed) GetUprStats() *UprStats {
	return &feed.stats
}

func composeOpaque(vbno, opaqueMSB uint16) uint32 {
	return (uint32(opaqueMSB) << 16) | uint32(vbno)
}

func appOpaque(opq32 uint32) uint16 {
	return uint16((opq32 & 0xFFFF0000) >> 16)
}

func vbOpaque(opq32 uint32) uint16 {
	return uint16(opq32 & 0xFFFF)
}

// Close this UprFeed.
func (feed *UprFeed) Close() {
	close(feed.closer)
}
