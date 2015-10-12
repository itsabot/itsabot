package couchbase

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
)

// Bucket auto-updater gets the latest version of the bucket config from
// the server. If the configuration has changed then updated the local
// bucket information. If the bucket has been deleted then notify anyone
// who is holding a reference to this bucket

const MAX_RETRY_COUNT = 5
const DISCONNECT_PERIOD = 120 * time.Second

type NotifyFn func(bucket string, err error)

// Use TCP keepalive to detect half close sockets
var updaterTransport http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	Dial: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial,
}

var updaterHTTPClient = &http.Client{Transport: updaterTransport}

func doHTTPRequestForUpdate(req *http.Request) (*http.Response, error) {

	var err error
	var res *http.Response

	for i := 0; i < HTTP_MAX_RETRY; i++ {
		res, err = updaterHTTPClient.Do(req)
		if err != nil && isHttpConnError(err) {
			continue
		}
		break
	}

	if err != nil {
		log.Printf(" HTTP request returned error %v", err)
		return nil, err
	}

	return res, err
}

func (b *Bucket) RunBucketUpdater(notify NotifyFn) {
	go func() {
		err := b.UpdateBucket()
		if err != nil {
			if notify != nil {
				notify(b.Name, err)
			}
			log.Printf(" Bucket Updater exited with err %v", err)
		}
	}()
}

// this version of replaceConnPools will not close the pools that are being reused
func (b *Bucket) replaceConnPools2(with []*connectionPool) {
	for {
		old := atomic.LoadPointer(&b.connPools)
		if atomic.CompareAndSwapPointer(&b.connPools, old, unsafe.Pointer(&with)) {
			if old != nil {
				for _, pool := range *(*[]*connectionPool)(old) {
					if pool != nil && pool.inUse == false {
						pool.Close()
					}
				}
			}
			return
		}
	}
}

func (b *Bucket) UpdateBucket() error {

	var failures int
	var returnErr error

	for {

		if failures == MAX_RETRY_COUNT {
			log.Printf(" Maximum failures reached. Exiting loop...")
			return fmt.Errorf("Max failures reached. Last Error %v", returnErr)
		}

		nodes := b.Nodes()
		if len(nodes) < 1 {
			return fmt.Errorf("No healthy nodes found")
		}

		startNode := rand.Intn(len(nodes))
		node := nodes[(startNode)%len(nodes)]

		streamUrl := fmt.Sprintf("http://%s/pools/default/bucketsStreaming/%s", node.Hostname, b.Name)
		log.Printf(" Trying with %s", streamUrl)
		req, err := http.NewRequest("GET", streamUrl, nil)
		if err != nil {
			return err
		}

		err = maybeAddAuth(req, b.pool.client.ah)
		if err != nil {
			return err
		}

		res, err := doHTTPRequestForUpdate(req)
		if err != nil {
			return err
		}

		if res.StatusCode != 200 {
			bod, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
			log.Printf("Failed to connect to host, unexpected status code: %v. Body %s", res.StatusCode, bod)
			res.Body.Close()
			returnErr = fmt.Errorf("Failed to connect to host. Status %v Body %s", res.StatusCode, bod)
			failures++
			continue
		}

		dec := json.NewDecoder(res.Body)

		tmpb := &Bucket{}
		for {

			err := dec.Decode(&tmpb)
			if err != nil {
				log.Printf(" Unable to decode response %v", err)
				returnErr = err
				res.Body.Close()
				break
			}

			// if we got here, reset failure count
			failures = 0
			b.Lock()

			// mark all the old connection pools for deletion
			pools := b.getConnPools()
			for _, pool := range pools {
				if pool != nil {
					pool.inUse = false
				}
			}

			newcps := make([]*connectionPool, len(tmpb.VBSMJson.ServerList))
			for i := range newcps {
				// get the old connection pool and check if it is still valid
				pool := b.getConnPoolByHost(tmpb.VBSMJson.ServerList[i])
				if pool != nil && pool.inUse == false {
					// if the hostname and index is unchanged then reuse this pool
					newcps[i] = pool
					pool.inUse = true
					continue
				}
				// else create a new pool
				if b.ah != nil {
					newcps[i] = newConnectionPool(
						tmpb.VBSMJson.ServerList[i],
						b.ah, PoolSize, PoolOverflow)

				} else {
					newcps[i] = newConnectionPool(
						tmpb.VBSMJson.ServerList[i],
						b.authHandler(), PoolSize, PoolOverflow)
				}
			}

			b.replaceConnPools2(newcps)

			tmpb.ah = b.ah
			atomic.StorePointer(&b.vBucketServerMap, unsafe.Pointer(&tmpb.VBSMJson))
			atomic.StorePointer(&b.nodeList, unsafe.Pointer(&tmpb.NodesJSON))
			b.Unlock()

			log.Printf("Got new configuration for bucket %s", b.Name)

		}
		// we are here because of an error
		failures++
		continue

	}
	return nil
}
