//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

// Package cbdatasource streams data from a Couchbase cluster.  It is
// implemented using Couchbase DCP protocol and has auto-reconnecting
// and auto-restarting goroutines underneath the hood to provide a
// simple, high-level cluster-wide abstraction.  By using
// cbdatasource, your application does not need to worry about
// connections or reconnections to individual server nodes or cluster
// topology changes, rebalance & failovers.  The API starting point is
// NewBucketDataSource().
package cbdatasource

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/go-couchbase"
	"github.com/couchbase/gomemcached"
	"github.com/couchbase/gomemcached/client"
)

// BucketDataSource is the main control interface returned by
// NewBucketDataSource().
type BucketDataSource interface {
	// Use Start() to kickoff connectivity to a Couchbase cluster,
	// after which calls will be made to the Receiver's methods.
	Start() error

	// Asynchronously request a cluster map refresh.  A reason string
	// of "" is valid.
	Kick(reason string) error

	// Returns an immutable snapshot of stats.
	Stats(dest *BucketDataSourceStats) error

	// Stops the underlying goroutines.
	Close() error
}

// A Receiver interface is implemented by the application, or the
// receiver of data.  Calls to methods on this interface will be made
// by the BucketDataSource using multiple, concurrent goroutines, so
// the application should implement its own Receiver-side
// synchronizations if needed.
type Receiver interface {
	// Invoked in advisory fashion by the BucketDataSource when it
	// encounters an error.  The BucketDataSource will continue to try
	// to "heal" and restart connections, etc, as necessary.  The
	// Receiver has a recourse during these error notifications of
	// simply Close()'ing the BucketDataSource.
	OnError(error)

	// Invoked by the BucketDataSource when it has received a mutation
	// from the data source.  Receiver implementation is responsible
	// for making its own copies of the key and request.
	DataUpdate(vbucketID uint16, key []byte, seq uint64,
		r *gomemcached.MCRequest) error

	// Invoked by the BucketDataSource when it has received a deletion
	// or expiration from the data source.  Receiver implementation is
	// responsible for making its own copies of the key and request.
	DataDelete(vbucketID uint16, key []byte, seq uint64,
		r *gomemcached.MCRequest) error

	// An callback invoked by the BucketDataSource when it has
	// received a start snapshot message from the data source.  The
	// Receiver implementation, for example, might choose to optimize
	// persistence perhaps by preparing a batch write to
	// application-specific storage.
	SnapshotStart(vbucketID uint16, snapStart, snapEnd uint64, snapType uint32) error

	// The Receiver should persist the value parameter of
	// SetMetaData() for retrieval during some future call to
	// GetMetaData() by the BucketDataSource.  The metadata value
	// should be considered "in-stream", or as part of the sequence
	// history of mutations.  That is, a later Rollback() to some
	// previous sequence number for a particular vbucketID should
	// rollback both persisted metadata and regular data.
	SetMetaData(vbucketID uint16, value []byte) error

	// GetMetaData() should return the opaque value previously
	// provided by an earlier call to SetMetaData().  If there was no
	// previous call to SetMetaData(), such as in the case of a brand
	// new instance of a Receiver (as opposed to a restarted or
	// reloaded Receiver), the Receiver should return (nil, 0, nil)
	// for (value, lastSeq, err), respectively.  The lastSeq should be
	// the last sequence number received and persisted during calls to
	// the Receiver's DataUpdate() & DataDelete() methods.
	GetMetaData(vbucketID uint16) (value []byte, lastSeq uint64, err error)

	// Invoked by the BucketDataSource when the datasource signals a
	// rollback during stream initialization.  Note that both data and
	// metadata should be rolled back.
	Rollback(vbucketID uint16, rollbackSeq uint64) error
}

// BucketDataSourceOptions allows the application to provide
// configuration settings to NewBucketDataSource().
type BucketDataSourceOptions struct {
	// Optional - used during UPR_OPEN stream start.  If empty a
	// random name will be automatically generated.
	Name string

	// Factor (like 1.5) to increase sleep time between retries
	// in connecting to a cluster manager node.
	ClusterManagerBackoffFactor float32

	// Initial sleep time (millisecs) before first retry to cluster manager.
	ClusterManagerSleepInitMS int

	// Maximum sleep time (millisecs) between retries to cluster manager.
	ClusterManagerSleepMaxMS int

	// Factor (like 1.5) to increase sleep time between retries
	// in connecting to a data manager node.
	DataManagerBackoffFactor float32

	// Initial sleep time (millisecs) before first retry to data manager.
	DataManagerSleepInitMS int

	// Maximum sleep time (millisecs) between retries to data manager.
	DataManagerSleepMaxMS int

	// Buffer size in bytes provided for UPR flow control.
	FeedBufferSizeBytes uint32

	// Used for UPR flow control and buffer-ack messages when this
	// percentage of FeedBufferSizeBytes is reached.
	FeedBufferAckThreshold float32

	// Used for applications like backup which wish to control the
	// last sequence number provided.  Key is vbucketID, value is seqEnd.
	SeqEnd map[uint16]uint64

	// Optional function to connect to a couchbase cluster manager bucket.
	// Defaults to ConnectBucket() function in this package.
	ConnectBucket func(serverURL, poolName, bucketName string,
		auth couchbase.AuthHandler) (Bucket, error)

	// Optional function to connect to a couchbase data manager node.
	// Defaults to memcached.Connect().
	Connect func(protocol, dest string) (*memcached.Client, error)
}

// AllServerURLsConnectBucketError is the error type passed to
// Receiver.OnError() when the BucketDataSource failed to connect to
// all the serverURL's provided as a parameter to
// NewBucketDataSource().  The application, for example, may choose to
// BucketDataSource.Close() based on this error.  Otherwise, the
// BucketDataSource will backoff and retry reconnecting to the
// serverURL's.
type AllServerURLsConnectBucketError struct {
	ServerURLs []string
}

func (e *AllServerURLsConnectBucketError) Error() string {
	return fmt.Sprintf("could not connect to any serverURL: %#v", e.ServerURLs)
}

// AuthFailError is the error type passed to Receiver.OnError() when there
// is an auth request error to the Couchbase cluster or server node.
type AuthFailError struct {
	ServerURL string
	User      string
}

func (e *AuthFailError) Error() string {
	return fmt.Sprintf("auth fail, serverURL: %#v, user: %s", e.ServerURL, e.User)
}

// A Bucket interface defines the set of methods that cbdatasource
// needs from an abstract couchbase.Bucket.  This separate interface
// allows for easier testability.
type Bucket interface {
	Close()
	GetUUID() string
	VBServerMap() *couchbase.VBucketServerMap
}

// DefaultBucketDataSourceOptions defines the default options that
// will be used if nil is provided to NewBucketDataSource().
var DefaultBucketDataSourceOptions = &BucketDataSourceOptions{
	ClusterManagerBackoffFactor: 1.5,
	ClusterManagerSleepInitMS:   100,
	ClusterManagerSleepMaxMS:    1000,

	DataManagerBackoffFactor: 1.5,
	DataManagerSleepInitMS:   100,
	DataManagerSleepMaxMS:    1000,

	FeedBufferSizeBytes:    20000000, // ~20MB; see UPR_CONTROL/connection_buffer_size.
	FeedBufferAckThreshold: 0.2,
}

// BucketDataSourceStats is filled by the BucketDataSource.Stats()
// method.  All the metrics here prefixed with "Tot" are monotonic
// counters: they only increase.
type BucketDataSourceStats struct {
	TotStart  uint64
	TotKick   uint64
	TotKickOk uint64

	TotRefreshCluster                              uint64
	TotRefreshClusterConnectBucket                 uint64
	TotRefreshClusterConnectBucketErr              uint64
	TotRefreshClusterConnectBucketOk               uint64
	TotRefreshClusterBucketUUIDErr                 uint64
	TotRefreshClusterVBMNilErr                     uint64
	TotRefreshClusterKickWorkers                   uint64
	TotRefreshClusterKickWorkersOk                 uint64
	TotRefreshClusterAwokenClosed                  uint64
	TotRefreshClusterAwokenStopped                 uint64
	TotRefreshClusterAwokenRestart                 uint64
	TotRefreshClusterAwoken                        uint64
	TotRefreshClusterAllServerURLsConnectBucketErr uint64
	TotRefreshClusterDone                          uint64

	TotRefreshWorkers                uint64
	TotRefreshWorkersVBMNilErr       uint64
	TotRefreshWorkersVBucketIDErr    uint64
	TotRefreshWorkersServerIdxsErr   uint64
	TotRefreshWorkersMasterIdxErr    uint64
	TotRefreshWorkersMasterServerErr uint64
	TotRefreshWorkersRemoveWorker    uint64
	TotRefreshWorkersAddWorker       uint64
	TotRefreshWorkersKickWorker      uint64
	TotRefreshWorkersCloseWorker     uint64
	TotRefreshWorkersDone            uint64

	TotWorkerStart      uint64
	TotWorkerDone       uint64
	TotWorkerBody       uint64
	TotWorkerBodyKick   uint64
	TotWorkerConnect    uint64
	TotWorkerConnectErr uint64
	TotWorkerConnectOk  uint64
	TotWorkerAuth       uint64
	TotWorkerAuthErr    uint64
	TotWorkerAuthFail   uint64
	TotWorkerSelBktFail uint64
	TotWorkerSelBktOk   uint64
	TotWorkerAuthOk     uint64
	TotWorkerUPROpenErr uint64
	TotWorkerUPROpenOk  uint64

	TotWorkerTransmitStart uint64
	TotWorkerTransmit      uint64
	TotWorkerTransmitErr   uint64
	TotWorkerTransmitOk    uint64
	TotWorkerTransmitDone  uint64

	TotWorkerReceiveStart uint64
	TotWorkerReceive      uint64
	TotWorkerReceiveErr   uint64
	TotWorkerReceiveOk    uint64

	TotWorkerSendEndCh uint64
	TotWorkerRecvEndCh uint64

	TotWorkerHandleRecv    uint64
	TotWorkerHandleRecvErr uint64
	TotWorkerHandleRecvOk  uint64

	TotRefreshWorker     uint64
	TotRefreshWorkerDone uint64
	TotRefreshWorkerOk   uint64

	TotUPRDataChange                       uint64
	TotUPRDataChangeStateErr               uint64
	TotUPRDataChangeMutation               uint64
	TotUPRDataChangeDeletion               uint64
	TotUPRDataChangeExpiration             uint64
	TotUPRDataChangeErr                    uint64
	TotUPRDataChangeOk                     uint64
	TotUPRCloseStream                      uint64
	TotUPRCloseStreamRes                   uint64
	TotUPRCloseStreamResStateErr           uint64
	TotUPRCloseStreamResErr                uint64
	TotUPRCloseStreamResOk                 uint64
	TotUPRStreamReq                        uint64
	TotUPRStreamReqWant                    uint64
	TotUPRStreamReqRes                     uint64
	TotUPRStreamReqResStateErr             uint64
	TotUPRStreamReqResFail                 uint64
	TotUPRStreamReqResFailNotMyVBucket     uint64
	TotUPRStreamReqResFailERange           uint64
	TotUPRStreamReqResFailENoMem           uint64
	TotUPRStreamReqResRollback             uint64
	TotUPRStreamReqResRollbackStart        uint64
	TotUPRStreamReqResRollbackErr          uint64
	TotUPRStreamReqResWantAfterRollbackErr uint64
	TotUPRStreamReqResKick                 uint64
	TotUPRStreamReqResSuccess              uint64
	TotUPRStreamReqResSuccessOk            uint64
	TotUPRStreamReqResFLogErr              uint64
	TotUPRStreamEnd                        uint64
	TotUPRStreamEndStateErr                uint64
	TotUPRStreamEndKick                    uint64
	TotUPRSnapshot                         uint64
	TotUPRSnapshotStateErr                 uint64
	TotUPRSnapshotStart                    uint64
	TotUPRSnapshotStartErr                 uint64
	TotUPRSnapshotOk                       uint64
	TotUPRNoop                             uint64
	TotUPRControl                          uint64
	TotUPRControlErr                       uint64
	TotUPRBufferAck                        uint64

	TotWantCloseRequestedVBucketErr uint64
	TotWantClosingVBucketErr        uint64

	TotGetVBucketMetaData             uint64
	TotGetVBucketMetaDataUnmarshalErr uint64
	TotGetVBucketMetaDataErr          uint64
	TotGetVBucketMetaDataOk           uint64

	TotSetVBucketMetaData           uint64
	TotSetVBucketMetaDataMarshalErr uint64
	TotSetVBucketMetaDataErr        uint64
	TotSetVBucketMetaDataOk         uint64
}

// --------------------------------------------------------

// VBucketMetaData is an internal struct is exposed to enable json
// marshaling.
type VBucketMetaData struct {
	SeqStart    uint64     `json:"seqStart"`
	SeqEnd      uint64     `json:"seqEnd"`
	SnapStart   uint64     `json:"snapStart"`
	SnapEnd     uint64     `json:"snapEnd"`
	FailOverLog [][]uint64 `json:"failOverLog"`
}

type bucketDataSource struct {
	serverURLs []string
	poolName   string
	bucketName string
	bucketUUID string
	vbucketIDs []uint16
	auth       couchbase.AuthHandler // auth for couchbase
	receiver   Receiver
	options    *BucketDataSourceOptions

	refreshClusterCh chan string
	refreshWorkersCh chan string
	closedCh         chan bool

	stats BucketDataSourceStats

	m    sync.Mutex // Protects all the below fields.
	life string     // Valid life states: "" (unstarted); "running"; "closed".
	vbm  *couchbase.VBucketServerMap
}

// NewBucketDataSource is the main starting point for using the
// cbdatasource API.  The application must supply an array of 1 or
// more serverURLs (or "seed" URL's) to Couchbase Server
// cluster-manager REST URL endpoints, like "http://localhost:8091".
// The BucketDataSource (after Start()'ing) will try each serverURL,
// in turn, until it can get a successful cluster map.  Additionally,
// the application must supply a poolName & bucketName from where the
// BucketDataSource will retrieve data.  The optional bucketUUID is
// double-checked by the BucketDataSource to ensure we have the
// correct bucket, and a bucketUUID of "" means skip the bucketUUID
// validation.  An optional array of vbucketID numbers allows the
// application to specify which vbuckets to retrieve; and the
// vbucketIDs array can be nil which means all vbuckets are retrieved
// by the BucketDataSource.  The optional auth parameter can be nil.
// The application must supply its own implementation of the Receiver
// interface (see the example program as a sample).  The optional
// options parameter (which may be nil) allows the application to
// specify advanced parameters like backoff and retry-sleep values.
func NewBucketDataSource(
	serverURLs []string,
	poolName string,
	bucketName string,
	bucketUUID string,
	vbucketIDs []uint16,
	auth couchbase.AuthHandler,
	receiver Receiver,
	options *BucketDataSourceOptions) (BucketDataSource, error) {
	if len(serverURLs) < 1 {
		return nil, fmt.Errorf("missing at least 1 serverURL")
	}
	if poolName == "" {
		return nil, fmt.Errorf("missing poolName")
	}
	if bucketName == "" {
		return nil, fmt.Errorf("missing bucketName")
	}
	if receiver == nil {
		return nil, fmt.Errorf("missing receiver")
	}
	if options == nil {
		options = DefaultBucketDataSourceOptions
	}
	return &bucketDataSource{
		serverURLs: serverURLs,
		poolName:   poolName,
		bucketName: bucketName,
		bucketUUID: bucketUUID,
		vbucketIDs: vbucketIDs,
		auth:       auth,
		receiver:   receiver,
		options:    options,

		refreshClusterCh: make(chan string, 1),
		refreshWorkersCh: make(chan string, 1),
		closedCh:         make(chan bool),
	}, nil
}

func (d *bucketDataSource) Start() error {
	atomic.AddUint64(&d.stats.TotStart, 1)

	d.m.Lock()
	if d.life != "" {
		d.m.Unlock()
		return fmt.Errorf("call to Start() in wrong state: %s", d.life)
	}
	d.life = "running"
	d.m.Unlock()

	backoffFactor := d.options.ClusterManagerBackoffFactor
	if backoffFactor <= 0.0 {
		backoffFactor = DefaultBucketDataSourceOptions.ClusterManagerBackoffFactor
	}
	sleepInitMS := d.options.ClusterManagerSleepInitMS
	if sleepInitMS <= 0 {
		sleepInitMS = DefaultBucketDataSourceOptions.ClusterManagerSleepInitMS
	}
	sleepMaxMS := d.options.ClusterManagerSleepMaxMS
	if sleepMaxMS <= 0 {
		sleepMaxMS = DefaultBucketDataSourceOptions.ClusterManagerSleepMaxMS
	}

	go func() {
		ExponentialBackoffLoop("cbdatasource.refreshCluster",
			func() int { return d.refreshCluster() },
			sleepInitMS, backoffFactor, sleepMaxMS)

		// We reach here when we need to shutdown.
		close(d.refreshWorkersCh)
		atomic.AddUint64(&d.stats.TotRefreshClusterDone, 1)
	}()

	go d.refreshWorkers()

	return nil
}

func (d *bucketDataSource) isRunning() bool {
	d.m.Lock()
	life := d.life
	d.m.Unlock()
	return life == "running"
}

func (d *bucketDataSource) refreshCluster() int {
	atomic.AddUint64(&d.stats.TotRefreshCluster, 1)

	if !d.isRunning() {
		return -1
	}

	for _, serverURL := range d.serverURLs {
		atomic.AddUint64(&d.stats.TotRefreshClusterConnectBucket, 1)

		connectBucket := d.options.ConnectBucket
		if connectBucket == nil {
			connectBucket = ConnectBucket
		}

		bucket, err := connectBucket(serverURL, d.poolName, d.bucketName, d.auth)
		if err != nil {
			atomic.AddUint64(&d.stats.TotRefreshClusterConnectBucketErr, 1)
			d.receiver.OnError(err)
			continue // Try another serverURL.
		}
		atomic.AddUint64(&d.stats.TotRefreshClusterConnectBucketOk, 1)

		if d.bucketUUID != "" && d.bucketUUID != bucket.GetUUID() {
			bucket.Close()
			atomic.AddUint64(&d.stats.TotRefreshClusterBucketUUIDErr, 1)
			d.receiver.OnError(fmt.Errorf("mismatched bucket uuid,"+
				" serverURL: %s, bucketName: %s, bucketUUID: %s, bucket.UUID: %s",
				serverURL, d.bucketName, d.bucketUUID, bucket.GetUUID()))
			continue // Try another serverURL.
		}

		vbm := bucket.VBServerMap()
		if vbm == nil {
			bucket.Close()
			atomic.AddUint64(&d.stats.TotRefreshClusterVBMNilErr, 1)
			d.receiver.OnError(fmt.Errorf("refreshCluster got no vbm,"+
				" serverURL: %s, bucketName: %s, bucketUUID: %s, bucket.UUID: %s",
				serverURL, d.bucketName, d.bucketUUID, bucket.GetUUID()))
			continue // Try another serverURL.
		}

		bucket.Close()

		d.m.Lock()
		d.vbm = vbm
		d.m.Unlock()

		for {
			atomic.AddUint64(&d.stats.TotRefreshClusterKickWorkers, 1)
			d.refreshWorkersCh <- "new-vbm" // Kick the workers to refresh.
			atomic.AddUint64(&d.stats.TotRefreshClusterKickWorkersOk, 1)

			reason, alive := <-d.refreshClusterCh // Wait for a refresh cluster kick.
			if !alive || reason == "CLOSE" {      // Or, if we're closed then shutdown.
				atomic.AddUint64(&d.stats.TotRefreshClusterAwokenClosed, 1)
				return -1
			}
			if !d.isRunning() {
				atomic.AddUint64(&d.stats.TotRefreshClusterAwokenStopped, 1)
				return -1
			}

			// If it's only that a new worker appeared, then we can
			// keep with this inner loop and not have to restart all
			// the way at the top / retrieve a new cluster map, etc.
			if reason != "new-worker" {
				atomic.AddUint64(&d.stats.TotRefreshClusterAwokenRestart, 1)
				return 1 // Assume progress, so restart at first serverURL.
			}

			atomic.AddUint64(&d.stats.TotRefreshClusterAwoken, 1)
		}
	}

	// Notify Receiver in case it wants to Close() down this
	// BucketDataSource after enough attempts.  The typed interfaces
	// allow Receiver to have better error handling logic.
	atomic.AddUint64(&d.stats.TotRefreshClusterAllServerURLsConnectBucketErr, 1)
	d.receiver.OnError(&AllServerURLsConnectBucketError{ServerURLs: d.serverURLs})

	return 0 // Ran through all the serverURLs, so no progress.
}

func (d *bucketDataSource) refreshWorkers() {
	// Keyed by server, value is chan of array of vbucketID's that the
	// worker needs to provide.
	workers := make(map[string]chan []uint16)

	for _ = range d.refreshWorkersCh { // Wait for a refresh kick.
		atomic.AddUint64(&d.stats.TotRefreshWorkers, 1)

		d.m.Lock()
		vbm := d.vbm
		d.m.Unlock()

		if vbm == nil {
			atomic.AddUint64(&d.stats.TotRefreshWorkersVBMNilErr, 1)
			continue
		}

		// If nil vbucketIDs, then default to all vbucketIDs.
		vbucketIDs := d.vbucketIDs
		if vbucketIDs == nil {
			vbucketIDs = make([]uint16, len(vbm.VBucketMap))
			for i := 0; i < len(vbucketIDs); i++ {
				vbucketIDs[i] = uint16(i)
			}
		}

		// Group the wanted vbucketIDs by server.
		vbucketIDsByServer := make(map[string][]uint16)

		for _, vbucketID := range vbucketIDs {
			if int(vbucketID) >= len(vbm.VBucketMap) {
				atomic.AddUint64(&d.stats.TotRefreshWorkersVBucketIDErr, 1)
				d.receiver.OnError(fmt.Errorf("refreshWorkers"+
					" saw bad vbucketID: %d, vbm: %#v",
					vbucketID, vbm))
				continue
			}
			serverIdxs := vbm.VBucketMap[vbucketID]
			if serverIdxs == nil || len(serverIdxs) <= 0 {
				atomic.AddUint64(&d.stats.TotRefreshWorkersServerIdxsErr, 1)
				d.receiver.OnError(fmt.Errorf("refreshWorkers"+
					" no serverIdxs for vbucketID: %d, vbm: %#v",
					vbucketID, vbm))
				continue
			}
			masterIdx := serverIdxs[0]
			if int(masterIdx) >= len(vbm.ServerList) {
				atomic.AddUint64(&d.stats.TotRefreshWorkersMasterIdxErr, 1)
				d.receiver.OnError(fmt.Errorf("refreshWorkers"+
					" no masterIdx for vbucketID: %d, vbm: %#v",
					vbucketID, vbm))
				continue
			}
			masterServer := vbm.ServerList[masterIdx]
			if masterServer == "" {
				atomic.AddUint64(&d.stats.TotRefreshWorkersMasterServerErr, 1)
				d.receiver.OnError(fmt.Errorf("refreshWorkers"+
					" no masterServer for vbucketID: %d, vbm: %#v",
					vbucketID, vbm))
				continue
			}
			v, exists := vbucketIDsByServer[masterServer]
			if !exists || v == nil {
				v = []uint16{}
			}
			vbucketIDsByServer[masterServer] = append(v, vbucketID)
		}

		// Remove any extraneous workers.
		for server, workerCh := range workers {
			if _, exists := vbucketIDsByServer[server]; !exists {
				atomic.AddUint64(&d.stats.TotRefreshWorkersRemoveWorker, 1)
				delete(workers, server)
				close(workerCh)
			}
		}

		// Add any missing workers and update workers with their
		// latest vbucketIDs.
		for server, serverVBucketIDs := range vbucketIDsByServer {
			workerCh, exists := workers[server]
			if !exists || workerCh == nil {
				atomic.AddUint64(&d.stats.TotRefreshWorkersAddWorker, 1)
				workerCh = make(chan []uint16, 1)
				workers[server] = workerCh
				d.workerStart(server, workerCh)
			}

			workerCh <- serverVBucketIDs
			atomic.AddUint64(&d.stats.TotRefreshWorkersKickWorker, 1)
		}
	}

	// We reach here when we need to shutdown.
	for _, workerCh := range workers {
		atomic.AddUint64(&d.stats.TotRefreshWorkersCloseWorker, 1)
		close(workerCh)
	}

	close(d.closedCh)
	atomic.AddUint64(&d.stats.TotRefreshWorkersDone, 1)
}

// A worker connects to one data manager server.
func (d *bucketDataSource) workerStart(server string, workerCh chan []uint16) {
	backoffFactor := d.options.DataManagerBackoffFactor
	if backoffFactor <= 0.0 {
		backoffFactor = DefaultBucketDataSourceOptions.DataManagerBackoffFactor
	}
	sleepInitMS := d.options.DataManagerSleepInitMS
	if sleepInitMS <= 0 {
		sleepInitMS = DefaultBucketDataSourceOptions.DataManagerSleepInitMS
	}
	sleepMaxMS := d.options.DataManagerSleepMaxMS
	if sleepMaxMS <= 0 {
		sleepMaxMS = DefaultBucketDataSourceOptions.DataManagerSleepMaxMS
	}

	// Use exponential backoff loop to handle reconnect retries to the server.
	go func() {
		atomic.AddUint64(&d.stats.TotWorkerStart, 1)

		ExponentialBackoffLoop("cbdatasource.worker-"+server,
			func() int { return d.worker(server, workerCh) },
			sleepInitMS, backoffFactor, sleepMaxMS)

		atomic.AddUint64(&d.stats.TotWorkerDone, 1)
	}()
}

type VBucketState struct {
	// Valid values for state: "" (dead/closed/unknown), "requested",
	// "running", "closing".
	State     string
	SnapStart uint64
	SnapEnd   uint64
	SnapSaved bool // True when the snapStart/snapEnd have been persisted.
}

// Connect once to the server and work the UPR stream.  If anything
// goes wrong, return our level of progress in order to let our caller
// control any potential retries.
func (d *bucketDataSource) worker(server string, workerCh chan []uint16) int {
	atomic.AddUint64(&d.stats.TotWorkerBody, 1)

	if !d.isRunning() {
		return -1
	}

	atomic.AddUint64(&d.stats.TotWorkerConnect, 1)
	connect := d.options.Connect
	if connect == nil {
		connect = memcached.Connect
	}

	client, err := connect("tcp", server)
	if err != nil {
		atomic.AddUint64(&d.stats.TotWorkerConnectErr, 1)
		d.receiver.OnError(fmt.Errorf("worker connect, server: %s, err: %v",
			server, err))
		return 0
	}
	defer client.Close()
	atomic.AddUint64(&d.stats.TotWorkerConnectOk, 1)

	if d.auth != nil {
		var user, pswd string
		var adminCred bool
		if auth, ok := d.auth.(couchbase.AuthWithSaslHandler); ok {
			user, pswd = auth.GetSaslCredentials()
			adminCred = true
		} else {
			user, pswd, _ = d.auth.GetCredentials()
		}
		if user != "" {
			atomic.AddUint64(&d.stats.TotWorkerAuth, 1)
			res, err := client.Auth(user, pswd)
			if err != nil {
				atomic.AddUint64(&d.stats.TotWorkerAuthErr, 1)
				d.receiver.OnError(fmt.Errorf("worker auth, server: %s, user: %s, err: %v",
					server, user, err))
				return 0
			}
			if res.Status != gomemcached.SUCCESS {
				atomic.AddUint64(&d.stats.TotWorkerAuthFail, 1)
				d.receiver.OnError(&AuthFailError{ServerURL: server, User: user})
				return 0
			}
			if adminCred {
				atomic.AddUint64(&d.stats.TotWorkerAuthOk, 1)
				_, err = client.SelectBucket(d.bucketName)
				if err != nil {
					atomic.AddUint64(&d.stats.TotWorkerSelBktFail, 1)
					d.receiver.OnError(fmt.Errorf("worker select bucket err: %v", err))
					return 0
				}
				atomic.AddUint64(&d.stats.TotWorkerSelBktOk, 1)
			}
		}
	}

	uprOpenName := d.options.Name
	if uprOpenName == "" {
		uprOpenName = fmt.Sprintf("cbdatasource-%x", rand.Int63())
	}

	err = UPROpen(client, uprOpenName, d.options.FeedBufferSizeBytes)
	if err != nil {
		atomic.AddUint64(&d.stats.TotWorkerUPROpenErr, 1)
		d.receiver.OnError(err)
		return 0
	}
	atomic.AddUint64(&d.stats.TotWorkerUPROpenOk, 1)

	ackBytes :=
		uint32(d.options.FeedBufferAckThreshold * float32(d.options.FeedBufferSizeBytes))

	sendCh := make(chan *gomemcached.MCRequest, 1)
	sendEndCh := make(chan struct{})
	recvEndCh := make(chan struct{})

	cleanup := func(progress int, err error) int {
		if err != nil {
			d.receiver.OnError(err)
		}
		go func() {
			<-recvEndCh
			close(sendCh)
		}()
		return progress
	}

	currVBuckets := make(map[uint16]*VBucketState)
	currVBucketsMutex := sync.Mutex{} // Protects currVBuckets.

	go func() { // Sender goroutine.
		defer close(sendEndCh)

		atomic.AddUint64(&d.stats.TotWorkerTransmitStart, 1)
		for msg := range sendCh {
			atomic.AddUint64(&d.stats.TotWorkerTransmit, 1)
			err := client.Transmit(msg)
			if err != nil {
				atomic.AddUint64(&d.stats.TotWorkerTransmitErr, 1)
				d.receiver.OnError(fmt.Errorf("client.Transmit, err: %v", err))
				return
			}
			atomic.AddUint64(&d.stats.TotWorkerTransmitOk, 1)
		}
		atomic.AddUint64(&d.stats.TotWorkerTransmitDone, 1)
	}()

	go func() { // Receiver goroutine.
		defer close(recvEndCh)

		atomic.AddUint64(&d.stats.TotWorkerReceiveStart, 1)

		var hdr [gomemcached.HDR_LEN]byte
		var pkt gomemcached.MCRequest
		var res gomemcached.MCResponse

		// Track received bytes in case we need to buffer-ack.
		recvBytesTotal := uint32(0)

		conn := client.Hijack()

		for {
			// TODO: memory allocation here.
			atomic.AddUint64(&d.stats.TotWorkerReceive, 1)
			_, err := pkt.Receive(conn, hdr[:])
			if err != nil {
				atomic.AddUint64(&d.stats.TotWorkerReceiveErr, 1)
				d.receiver.OnError(fmt.Errorf("pkt.Receive, err: %v", err))
				return
			}
			atomic.AddUint64(&d.stats.TotWorkerReceiveOk, 1)

			if pkt.Opcode == gomemcached.UPR_MUTATION ||
				pkt.Opcode == gomemcached.UPR_DELETION ||
				pkt.Opcode == gomemcached.UPR_EXPIRATION {
				atomic.AddUint64(&d.stats.TotUPRDataChange, 1)

				vbucketID := pkt.VBucket

				currVBucketsMutex.Lock()

				vbucketState := currVBuckets[vbucketID]
				if vbucketState == nil || vbucketState.State != "running" {
					currVBucketsMutex.Unlock()
					atomic.AddUint64(&d.stats.TotUPRDataChangeStateErr, 1)
					d.receiver.OnError(fmt.Errorf("error: DataChange,"+
						" wrong vbucketState: %#v, err: %v", vbucketState, err))
					return
				}

				if !vbucketState.SnapSaved {
					// NOTE: Following the ep-engine's approach, we
					// wait to persist SnapStart/SnapEnd until we see
					// the first mutation/deletion in the new snapshot
					// range.  That reduces a race window where if we
					// kill and restart this process right now after a
					// setVBucketMetaData() and before the next,
					// first-mutation-in-snapshot, then a restarted
					// stream-req using this just-saved
					// SnapStart/SnapEnd might have a lastSeq number <
					// SnapStart, where Couchbase Server will respond
					// to the stream-req with an ERANGE error code.
					v, _, err := d.getVBucketMetaData(vbucketID)
					if err != nil || v == nil {
						currVBucketsMutex.Unlock()
						d.receiver.OnError(fmt.Errorf("error: DataChange,"+
							" getVBucketMetaData, vbucketID: %d, err: %v",
							vbucketID, err))
						return
					}

					v.SnapStart = vbucketState.SnapStart
					v.SnapEnd = vbucketState.SnapEnd

					err = d.setVBucketMetaData(vbucketID, v)
					if err != nil {
						currVBucketsMutex.Unlock()
						d.receiver.OnError(fmt.Errorf("error: DataChange,"+
							" getVBucketMetaData, vbucketID: %d, err: %v",
							vbucketID, err))
						return
					}

					vbucketState.SnapSaved = true
				}

				currVBucketsMutex.Unlock()

				seq := binary.BigEndian.Uint64(pkt.Extras[:8])

				if pkt.Opcode == gomemcached.UPR_MUTATION {
					atomic.AddUint64(&d.stats.TotUPRDataChangeMutation, 1)
					err = d.receiver.DataUpdate(vbucketID, pkt.Key, seq, &pkt)
				} else {
					if pkt.Opcode == gomemcached.UPR_DELETION {
						atomic.AddUint64(&d.stats.TotUPRDataChangeDeletion, 1)
					} else {
						atomic.AddUint64(&d.stats.TotUPRDataChangeExpiration, 1)
					}
					err = d.receiver.DataDelete(vbucketID, pkt.Key, seq, &pkt)
				}

				if err != nil {
					atomic.AddUint64(&d.stats.TotUPRDataChangeErr, 1)
					d.receiver.OnError(fmt.Errorf("error: DataChange, err: %v", err))
					return
				}

				atomic.AddUint64(&d.stats.TotUPRDataChangeOk, 1)
			} else {
				res.Opcode = pkt.Opcode
				res.Opaque = pkt.Opaque
				res.Status = gomemcached.Status(pkt.VBucket)
				res.Extras = pkt.Extras
				res.Cas = pkt.Cas
				res.Key = pkt.Key
				res.Body = pkt.Body

				atomic.AddUint64(&d.stats.TotWorkerHandleRecv, 1)
				currVBucketsMutex.Lock()
				err := d.handleRecv(sendCh, currVBuckets, &res)
				currVBucketsMutex.Unlock()
				if err != nil {
					atomic.AddUint64(&d.stats.TotWorkerHandleRecvErr, 1)
					d.receiver.OnError(fmt.Errorf("error: HandleRecv, err: %v", err))
					return
				}
				atomic.AddUint64(&d.stats.TotWorkerHandleRecvOk, 1)
			}

			recvBytesTotal +=
				uint32(gomemcached.HDR_LEN) +
					uint32(len(pkt.Key)+len(pkt.Extras)+len(pkt.Body))
			if ackBytes > 0 && recvBytesTotal > ackBytes {
				atomic.AddUint64(&d.stats.TotUPRBufferAck, 1)
				ack := &gomemcached.MCRequest{Opcode: gomemcached.UPR_BUFFERACK}
				ack.Extras = make([]byte, 4) // TODO: Memory mgmt.
				binary.BigEndian.PutUint32(ack.Extras, uint32(recvBytesTotal))
				sendCh <- ack
				recvBytesTotal = 0
			}
		}
	}()

	atomic.AddUint64(&d.stats.TotWorkerBodyKick, 1)
	d.Kick("new-worker")

	for {
		select {
		case <-sendEndCh:
			atomic.AddUint64(&d.stats.TotWorkerSendEndCh, 1)
			return cleanup(0, nil)

		case <-recvEndCh:
			// If we lost a connection, then maybe a node was rebalanced out,
			// or failed over, so ask for a cluster refresh just in case.
			d.Kick("recvEndCh")

			atomic.AddUint64(&d.stats.TotWorkerRecvEndCh, 1)
			return cleanup(0, nil)

		case wantVBucketIDs, alive := <-workerCh:
			atomic.AddUint64(&d.stats.TotRefreshWorker, 1)

			if !alive {
				atomic.AddUint64(&d.stats.TotRefreshWorkerDone, 1)
				return cleanup(-1, nil) // We've been asked to shutdown.
			}

			currVBucketsMutex.Lock()
			err := d.refreshWorker(sendCh, currVBuckets, wantVBucketIDs)
			currVBucketsMutex.Unlock()
			if err != nil {
				return cleanup(0, err)
			}

			atomic.AddUint64(&d.stats.TotRefreshWorkerOk, 1)
		}
	}

	return cleanup(-1, nil) // Unreached.
}

func (d *bucketDataSource) refreshWorker(sendCh chan *gomemcached.MCRequest,
	currVBuckets map[uint16]*VBucketState, wantVBucketIDsArr []uint16) error {
	// Convert to map for faster lookup.
	wantVBucketIDs := map[uint16]bool{}
	for _, wantVBucketID := range wantVBucketIDsArr {
		wantVBucketIDs[wantVBucketID] = true
	}

	for currVBucketID, state := range currVBuckets {
		if !wantVBucketIDs[currVBucketID] {
			if state != nil {
				if state.State == "requested" {
					// A UPR_STREAMREQ request is already on the wire, so
					// error rather than have complex compensation logic.
					atomic.AddUint64(&d.stats.TotWantCloseRequestedVBucketErr, 1)
					return fmt.Errorf("want close requested vbucketID: %d", currVBucketID)
				}
				if state.State == "running" {
					state.State = "closing"
					atomic.AddUint64(&d.stats.TotUPRCloseStream, 1)
					sendCh <- &gomemcached.MCRequest{
						Opcode:  gomemcached.UPR_CLOSESTREAM,
						VBucket: currVBucketID,
						Opaque:  uint32(currVBucketID),
					}
				} // Else, state.State of "" or "closing", so no-op.
			} // Else state of nil, so no-op.
		}
	}

	for wantVBucketID := range wantVBucketIDs {
		state := currVBuckets[wantVBucketID]
		if state != nil && state.State == "closing" {
			// A UPR_CLOSESTREAM request is already on the wire, so
			// error rather than have complex compensation logic.
			atomic.AddUint64(&d.stats.TotWantClosingVBucketErr, 1)
			return fmt.Errorf("want closing vbucketID: %d", wantVBucketID)
		}
		if state == nil || state.State == "" {
			currVBuckets[wantVBucketID] = &VBucketState{State: "requested"}
			atomic.AddUint64(&d.stats.TotUPRStreamReqWant, 1)
			err := d.sendStreamReq(sendCh, wantVBucketID)
			if err != nil {
				return err
			}
		} // Else, state.State of "requested" or "running", so no-op.
	}

	return nil
}

func (d *bucketDataSource) handleRecv(sendCh chan *gomemcached.MCRequest,
	currVBuckets map[uint16]*VBucketState, res *gomemcached.MCResponse) error {
	switch res.Opcode {
	case gomemcached.UPR_NOOP:
		atomic.AddUint64(&d.stats.TotUPRNoop, 1)
		sendCh <- &gomemcached.MCRequest{
			Opcode: gomemcached.UPR_NOOP,
			Opaque: res.Opaque,
		}

	case gomemcached.UPR_STREAMREQ:
		atomic.AddUint64(&d.stats.TotUPRStreamReqRes, 1)

		vbucketID := uint16(res.Opaque)
		vbucketState := currVBuckets[vbucketID]

		delete(currVBuckets, vbucketID)

		if vbucketState == nil || vbucketState.State != "requested" {
			atomic.AddUint64(&d.stats.TotUPRStreamReqResStateErr, 1)
			return fmt.Errorf("streamreq non-requested,"+
				" vbucketID: %d, vbucketState: %#v, res: %#v",
				vbucketID, vbucketState, res)
		}

		if res.Status != gomemcached.SUCCESS {
			atomic.AddUint64(&d.stats.TotUPRStreamReqResFail, 1)

			if res.Status == gomemcached.ROLLBACK ||
				res.Status == gomemcached.ERANGE {
				rollbackSeq := uint64(0)

				if res.Status == gomemcached.ROLLBACK {
					atomic.AddUint64(&d.stats.TotUPRStreamReqResRollback, 1)

					if len(res.Body) < 8 {
						return fmt.Errorf("bad rollback body: %#v", res)
					}

					rollbackSeq = binary.BigEndian.Uint64(res.Body)
				} else {
					// NOTE: Not sure what else to do here on ERANGE
					// error response besides rollback to zero.
					atomic.AddUint64(&d.stats.TotUPRStreamReqResFailERange, 1)
				}

				atomic.AddUint64(&d.stats.TotUPRStreamReqResRollbackStart, 1)
				err := d.receiver.Rollback(vbucketID, rollbackSeq)
				if err != nil {
					atomic.AddUint64(&d.stats.TotUPRStreamReqResRollbackErr, 1)
					return err
				}

				currVBuckets[vbucketID] = &VBucketState{State: "requested"}
				atomic.AddUint64(&d.stats.TotUPRStreamReqResWantAfterRollbackErr, 1)
				err = d.sendStreamReq(sendCh, vbucketID)
				if err != nil {
					return err
				}
			} else {
				if res.Status == gomemcached.NOT_MY_VBUCKET {
					atomic.AddUint64(&d.stats.TotUPRStreamReqResFailNotMyVBucket, 1)
				} else if res.Status == gomemcached.ENOMEM {
					atomic.AddUint64(&d.stats.TotUPRStreamReqResFailENoMem, 1)
				}

				// Maybe the vbucket moved, so kick off a cluster refresh.
				atomic.AddUint64(&d.stats.TotUPRStreamReqResKick, 1)
				d.Kick("stream-req-error")
			}
		} else { // SUCCESS case.
			atomic.AddUint64(&d.stats.TotUPRStreamReqResSuccess, 1)

			flog, err := ParseFailOverLog(res.Body[:])
			if err != nil {
				atomic.AddUint64(&d.stats.TotUPRStreamReqResFLogErr, 1)
				return err
			}
			v, _, err := d.getVBucketMetaData(vbucketID)
			if err != nil {
				return err
			}

			v.FailOverLog = flog

			err = d.setVBucketMetaData(vbucketID, v)
			if err != nil {
				return err
			}

			currVBuckets[vbucketID] = &VBucketState{State: "running"}
			atomic.AddUint64(&d.stats.TotUPRStreamReqResSuccessOk, 1)
		}

	case gomemcached.UPR_STREAMEND:
		atomic.AddUint64(&d.stats.TotUPRStreamEnd, 1)

		vbucketID := uint16(res.Status)
		vbucketState := currVBuckets[vbucketID]

		delete(currVBuckets, vbucketID)

		if vbucketState == nil ||
			(vbucketState.State != "running" && vbucketState.State != "closing") {
			atomic.AddUint64(&d.stats.TotUPRStreamEndStateErr, 1)
			return fmt.Errorf("stream-end bad state,"+
				" vbucketID: %d, vbucketState: %#v, res: %#v",
				vbucketID, vbucketState, res)
		}

		// We should not normally see a stream-end, unless we were
		// trying to close.  Maybe the vbucket moved, though, so kick
		// off a cluster refresh.
		if vbucketState.State != "closing" {
			atomic.AddUint64(&d.stats.TotUPRStreamEndKick, 1)
			d.Kick("stream-end")
		}

	case gomemcached.UPR_CLOSESTREAM:
		atomic.AddUint64(&d.stats.TotUPRCloseStreamRes, 1)

		vbucketID := uint16(res.Opaque)
		vbucketState := currVBuckets[vbucketID]

		if vbucketState == nil || vbucketState.State != "closing" {
			atomic.AddUint64(&d.stats.TotUPRCloseStreamResStateErr, 1)
			return fmt.Errorf("close-stream bad state,"+
				" vbucketID: %d, vbucketState: %#v, res: %#v",
				vbucketID, vbucketState, res)
		}

		if res.Status != gomemcached.SUCCESS {
			atomic.AddUint64(&d.stats.TotUPRCloseStreamResErr, 1)
			return fmt.Errorf("close-stream failed,"+
				" vbucketID: %d, vbucketState: %#v, res: %#v",
				vbucketID, vbucketState, res)
		}

		// At this point, we can ignore this success response to our
		// close-stream request, as the server will send a stream-end
		// afterwards.
		atomic.AddUint64(&d.stats.TotUPRCloseStreamResOk, 1)

	case gomemcached.UPR_SNAPSHOT:
		atomic.AddUint64(&d.stats.TotUPRSnapshot, 1)

		vbucketID := uint16(res.Status)
		vbucketState := currVBuckets[vbucketID]

		if vbucketState == nil || vbucketState.State != "running" {
			atomic.AddUint64(&d.stats.TotUPRSnapshotStateErr, 1)
			return fmt.Errorf("snapshot non-running,"+
				" vbucketID: %d, vbucketState: %#v, res: %#v",
				vbucketID, vbucketState, res)
		}

		if len(res.Extras) < 20 {
			return fmt.Errorf("bad snapshot extras, res: %#v", res)
		}

		vbucketState.SnapStart = binary.BigEndian.Uint64(res.Extras[0:8])
		vbucketState.SnapEnd = binary.BigEndian.Uint64(res.Extras[8:16])
		vbucketState.SnapSaved = false

		snapType := binary.BigEndian.Uint32(res.Extras[16:20])

		// NOTE: We should never see a snapType with SNAP_ACK flag of
		// true, as that's only used during takeovers, so that's why
		// we're not implementing SNAP_ACK handling here.

		atomic.AddUint64(&d.stats.TotUPRSnapshotStart, 1)
		err := d.receiver.SnapshotStart(vbucketID,
			vbucketState.SnapStart, vbucketState.SnapEnd, snapType)
		if err != nil {
			atomic.AddUint64(&d.stats.TotUPRSnapshotStartErr, 1)
			return err
		}

		atomic.AddUint64(&d.stats.TotUPRSnapshotOk, 1)

	case gomemcached.UPR_CONTROL:
		atomic.AddUint64(&d.stats.TotUPRControl, 1)
		if res.Status != gomemcached.SUCCESS {
			atomic.AddUint64(&d.stats.TotUPRControlErr, 1)
			return fmt.Errorf("failed control: %#v", res)
		}

	case gomemcached.UPR_OPEN:
		// Opening was long ago, so we should not see UPR_OPEN responses.
		return fmt.Errorf("unexpected upr_open, res: %#v", res)

	case gomemcached.UPR_ADDSTREAM:
		// This normally comes from ns-server / dcp-migrator.
		return fmt.Errorf("unexpected upr_addstream, res: %#v", res)

	case gomemcached.UPR_BUFFERACK:
		// We should be emitting buffer-ack's, not receiving them.
		return fmt.Errorf("unexpected buffer-ack, res: %#v", res)

	case gomemcached.UPR_MUTATION, gomemcached.UPR_DELETION, gomemcached.UPR_EXPIRATION:
		// This should have been handled already in receiver goroutine.
		return fmt.Errorf("unexpected data change, res: %#v", res)

	default:
		return fmt.Errorf("unknown opcode, res: %#v", res)
	}

	return nil
}

func (d *bucketDataSource) getVBucketMetaData(vbucketID uint16) (
	*VBucketMetaData, uint64, error) {
	atomic.AddUint64(&d.stats.TotGetVBucketMetaData, 1)

	buf, lastSeq, err := d.receiver.GetMetaData(vbucketID)
	if err != nil {
		atomic.AddUint64(&d.stats.TotGetVBucketMetaDataErr, 1)
		return nil, 0, err
	}

	vbucketMetaData := &VBucketMetaData{}
	if len(buf) > 0 {
		if err = json.Unmarshal(buf, vbucketMetaData); err != nil {
			atomic.AddUint64(&d.stats.TotGetVBucketMetaDataUnmarshalErr, 1)
			return nil, 0, err
		}
	}

	atomic.AddUint64(&d.stats.TotGetVBucketMetaDataOk, 1)
	return vbucketMetaData, lastSeq, nil
}

func (d *bucketDataSource) setVBucketMetaData(vbucketID uint16,
	v *VBucketMetaData) error {
	atomic.AddUint64(&d.stats.TotSetVBucketMetaData, 1)

	buf, err := json.Marshal(v)
	if err != nil {
		atomic.AddUint64(&d.stats.TotSetVBucketMetaDataMarshalErr, 1)
		return err
	}

	err = d.receiver.SetMetaData(vbucketID, buf)
	if err != nil {
		atomic.AddUint64(&d.stats.TotSetVBucketMetaDataErr, 1)
		return err
	}

	atomic.AddUint64(&d.stats.TotSetVBucketMetaDataOk, 1)
	return nil
}

func (d *bucketDataSource) sendStreamReq(sendCh chan *gomemcached.MCRequest,
	vbucketID uint16) error {
	vbucketMetaData, lastSeq, err := d.getVBucketMetaData(vbucketID)
	if err != nil {
		return fmt.Errorf("sendStreamReq, err: %v", err)
	}

	vbucketUUID := uint64(0)
	if len(vbucketMetaData.FailOverLog) >= 1 {
		smax := uint64(0)
		for _, pair := range vbucketMetaData.FailOverLog {
			if smax <= pair[1] {
				smax = pair[1]
				vbucketUUID = pair[0]
			}
		}
	}

	seqStart := lastSeq

	seqEnd := uint64(0xffffffffffffffff)
	if d.options.SeqEnd != nil { // Allow apps like backup to control the seqEnd.
		if s, exists := d.options.SeqEnd[vbucketID]; exists {
			seqEnd = s
		}
	}

	flags := uint32(0) // Flags mostly used for takeovers, etc, which we don't use.

	req := &gomemcached.MCRequest{
		Opcode:  gomemcached.UPR_STREAMREQ,
		VBucket: vbucketID,
		Opaque:  uint32(vbucketID),
		Extras:  make([]byte, 48),
	}
	binary.BigEndian.PutUint32(req.Extras[:4], flags)
	binary.BigEndian.PutUint32(req.Extras[4:8], uint32(0)) // Reserved.
	binary.BigEndian.PutUint64(req.Extras[8:16], seqStart)
	binary.BigEndian.PutUint64(req.Extras[16:24], seqEnd)
	binary.BigEndian.PutUint64(req.Extras[24:32], vbucketUUID)
	binary.BigEndian.PutUint64(req.Extras[32:40], vbucketMetaData.SnapStart)
	binary.BigEndian.PutUint64(req.Extras[40:48], vbucketMetaData.SnapEnd)

	atomic.AddUint64(&d.stats.TotUPRStreamReq, 1)
	sendCh <- req

	return nil
}

func (d *bucketDataSource) Stats(dest *BucketDataSourceStats) error {
	d.stats.AtomicCopyTo(dest, nil)
	return nil
}

func (d *bucketDataSource) Close() error {
	d.m.Lock()
	if d.life != "running" {
		d.m.Unlock()
		return fmt.Errorf("call to Close() when not running state: %s", d.life)
	}
	d.life = "closed"
	d.m.Unlock()

	// A close message to refreshClusterCh's goroutine will end
	// refreshWorkersCh's goroutine, which closes every workerCh and
	// then finally closes the closedCh.
	d.refreshClusterCh <- "CLOSE"

	<-d.closedCh

	// TODO: By this point, worker goroutines may still be going, but
	// should end soon.  Instead Close() should be 100% synchronous.
	return nil
}

func (d *bucketDataSource) Kick(reason string) error {
	go func() {
		if d.isRunning() {
			atomic.AddUint64(&d.stats.TotKick, 1)
			d.refreshClusterCh <- reason
			atomic.AddUint64(&d.stats.TotKickOk, 1)
		}
	}()

	return nil
}

// --------------------------------------------------------------

type bucketWrapper struct {
	b *couchbase.Bucket
}

func (bw *bucketWrapper) Close() {
	bw.b.Close()
}

func (bw *bucketWrapper) GetUUID() string {
	return bw.b.UUID
}

func (bw *bucketWrapper) VBServerMap() *couchbase.VBucketServerMap {
	return bw.b.VBServerMap()
}

// ConnectBucket is the default function used by BucketDataSource
// to connect to a Couchbase cluster to retrieve Bucket information.
// It is exposed for testability and to allow applications to
// override or wrap via BucketDataSourceOptions.
func ConnectBucket(serverURL, poolName, bucketName string,
	auth couchbase.AuthHandler) (Bucket, error) {
	var bucket *couchbase.Bucket
	var err error

	if auth != nil {
		client, err := couchbase.ConnectWithAuth(serverURL, auth)
		if err != nil {
			return nil, err
		}

		pool, err := client.GetPool(poolName)
		if err != nil {
			return nil, err
		}

		bucket, err = pool.GetBucket(bucketName)
	} else {
		bucket, err = couchbase.GetBucket(serverURL, poolName, bucketName)
	}
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, fmt.Errorf("unknown bucket,"+
			" serverURL: %s, bucketName: %s", serverURL, bucketName)
	}

	return &bucketWrapper{b: bucket}, nil
}

// UPROpen starts a UPR_OPEN stream on a memcached client connection.
// It is exposed for testability.
func UPROpen(mc *memcached.Client, name string, bufSize uint32) error {
	rq := &gomemcached.MCRequest{
		Opcode: gomemcached.UPR_OPEN,
		Key:    []byte(name),
		Opaque: 0xf00d1234,
		Extras: make([]byte, 8),
	}
	binary.BigEndian.PutUint32(rq.Extras[:4], 0) // First 4 bytes are reserved.
	flags := uint32(1)                           // NOTE: 1 for producer, 0 for consumer.
	binary.BigEndian.PutUint32(rq.Extras[4:], flags)

	if err := mc.Transmit(rq); err != nil {
		return fmt.Errorf("UPROpen transmit, err: %v", err)
	}
	res, err := mc.Receive()
	if err != nil {
		return fmt.Errorf("UPROpen receive, err: %v", err)
	}
	if res.Opcode != gomemcached.UPR_OPEN {
		return fmt.Errorf("UPROpen unexpected #opcode %v", res.Opcode)
	}
	if res.Opaque != rq.Opaque {
		return fmt.Errorf("UPROpen opaque mismatch, %v over %v", res.Opaque, res.Opaque)
	}
	if res.Status != gomemcached.SUCCESS {
		return fmt.Errorf("UPROpen failed, status: %v, %#v", res.Status, res)
	}
	if bufSize > 0 {
		rq := &gomemcached.MCRequest{
			Opcode: gomemcached.UPR_CONTROL,
			Key:    []byte("connection_buffer_size"),
			Body:   []byte(strconv.Itoa(int(bufSize))),
		}
		if err = mc.Transmit(rq); err != nil {
			return fmt.Errorf("UPROpen transmit UPR_CONTROL, err: %v", err)
		}
	}
	return nil
}

// ParseFailOverLog parses a byte array to an array of [vbucketUUID,
// seqNum] pairs.  It is exposed for testability.
func ParseFailOverLog(body []byte) ([][]uint64, error) {
	if len(body)%16 != 0 {
		return nil, fmt.Errorf("invalid body length %v, in failover-log", len(body))
	}
	flog := make([][]uint64, len(body)/16)
	for i, j := 0, 0; i < len(body); i += 16 {
		uuid := binary.BigEndian.Uint64(body[i : i+8])
		seqn := binary.BigEndian.Uint64(body[i+8 : i+16])
		flog[j] = []uint64{uuid, seqn}
		j++
	}
	return flog, nil
}

// --------------------------------------------------------------

// AtomicCopyTo copies metrics from s to r (or, from source to
// result), and also applies an optional fn function.  The fn is
// invoked with metrics from s and r, and can be used to compute
// additions, subtractions, negations, etc.  When fn is nil,
// AtomicCopyTo behaves as a straight copier.
func (s *BucketDataSourceStats) AtomicCopyTo(r *BucketDataSourceStats,
	fn func(sv uint64, rv uint64) uint64) {
	// Using reflection rather than a whole slew of explicit
	// invocations of atomic.LoadUint64()/StoreUint64()'s.
	if fn == nil {
		fn = func(sv uint64, rv uint64) uint64 { return sv }
	}
	rve := reflect.ValueOf(r).Elem()
	sve := reflect.ValueOf(s).Elem()
	svet := sve.Type()
	for i := 0; i < svet.NumField(); i++ {
		rvef := rve.Field(i)
		svef := sve.Field(i)
		if rvef.CanAddr() && svef.CanAddr() {
			rvefp := rvef.Addr().Interface()
			svefp := svef.Addr().Interface()
			rv := atomic.LoadUint64(rvefp.(*uint64))
			sv := atomic.LoadUint64(svefp.(*uint64))
			atomic.StoreUint64(rvefp.(*uint64), fn(sv, rv))
		}
	}
}

// --------------------------------------------------------------

// ExponentialBackoffLoop invokes f() in a loop, sleeping in an
// exponential number of milliseconds in between invocations if
// needed.  The provided f() function should return < 0 to stop the
// loop; >= 0 to continue the loop, where > 0 means there was progress
// which allows an immediate retry of f() with no sleeping.  A return
// of < 0 is useful when f() will never make any future progress.
// Repeated attempts with no progress will have exponential backoff
// sleep times.
func ExponentialBackoffLoop(name string,
	f func() int,
	startSleepMS int,
	backoffFactor float32,
	maxSleepMS int) {
	nextSleepMS := startSleepMS
	for {
		progress := f()
		if progress < 0 {
			return
		}
		if progress > 0 {
			// When there was some progress, we can reset nextSleepMS.
			nextSleepMS = startSleepMS
		} else {
			// If zero progress was made this cycle, then sleep.
			time.Sleep(time.Duration(nextSleepMS) * time.Millisecond)

			// Increase nextSleepMS in case next time also has 0 progress.
			nextSleepMS = int(float32(nextSleepMS) * backoffFactor)
			if nextSleepMS > maxSleepMS {
				nextSleepMS = maxSleepMS
			}
		}
	}
}
