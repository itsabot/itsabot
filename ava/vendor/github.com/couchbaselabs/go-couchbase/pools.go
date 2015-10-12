package couchbase

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	atomic "github.com/couchbase/go-couchbase/platform"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unsafe"

	"github.com/couchbase/gomemcached/client" // package name is 'memcached'
)

// HTTPClient to use for REST and view operations.
var MaxIdleConnsPerHost = 256
var HTTPTransport = &http.Transport{MaxIdleConnsPerHost: MaxIdleConnsPerHost}
var HTTPClient = &http.Client{Transport: HTTPTransport}

// PoolSize is the size of each connection pool (per host).
var PoolSize = 64

// PoolOverflow is the number of overflow connections allowed in a
// pool.
var PoolOverflow = 16

// TCP KeepAlive enabled/disabled
var TCPKeepalive = false

// TCP keepalive interval in seconds. Default 30 minutes
var TCPKeepaliveInterval = 30 * 60

// Allow applications to speciify the Poolsize and Overflow
func SetConnectionPoolParams(size, overflow int) {

	if size > 0 {
		PoolSize = size
	}

	if overflow > 0 {
		PoolOverflow = overflow
	}
}

// Allow TCP keepalive parameters to be set by the application
func SetTcpKeepalive(enabled bool, interval int) {

	TCPKeepalive = enabled

	if interval > 0 {
		TCPKeepaliveInterval = interval
	}
}

// AuthHandler is a callback that gets the auth username and password
// for the given bucket.
type AuthHandler interface {
	GetCredentials() (string, string, string)
}

// AuthHandler is a callback that gets the auth username and password
// for the given bucket and sasl for memcached.
type AuthWithSaslHandler interface {
	AuthHandler
	GetSaslCredentials() (string, string)
}

// MultiBucketAuthHandler is kind of AuthHandler that may perform
// different auth for different buckets.
type MultiBucketAuthHandler interface {
	AuthHandler
	ForBucket(bucket string) AuthHandler
}

// HTTPAuthHandler is kind of AuthHandler that performs more general
// for outgoing http requests than is possible via simple
// GetCredentials() call (i.e. digest auth or different auth per
// different destinations).
type HTTPAuthHandler interface {
	AuthHandler
	SetCredsForRequest(req *http.Request) error
}

// RestPool represents a single pool returned from the pools REST API.
type RestPool struct {
	Name         string `json:"name"`
	StreamingURI string `json:"streamingUri"`
	URI          string `json:"uri"`
}

// Pools represents the collection of pools as returned from the REST API.
type Pools struct {
	ComponentsVersion     map[string]string `json:"componentsVersion,omitempty"`
	ImplementationVersion string            `json:"implementationVersion"`
	IsAdmin               bool              `json:"isAdminCreds"`
	UUID                  string            `json:"uuid"`
	Pools                 []RestPool        `json:"pools"`
}

// A Node is a computer in a cluster running the couchbase software.
type Node struct {
	ClusterCompatibility int                `json:"clusterCompatibility"`
	ClusterMembership    string             `json:"clusterMembership"`
	CouchAPIBase         string             `json:"couchApiBase"`
	Hostname             string             `json:"hostname"`
	InterestingStats     map[string]float64 `json:"interestingStats,omitempty"`
	MCDMemoryAllocated   float64            `json:"mcdMemoryAllocated"`
	MCDMemoryReserved    float64            `json:"mcdMemoryReserved"`
	MemoryFree           float64            `json:"memoryFree"`
	MemoryTotal          float64            `json:"memoryTotal"`
	OS                   string             `json:"os"`
	Ports                map[string]int     `json:"ports"`
	Status               string             `json:"status"`
	Uptime               int                `json:"uptime,string"`
	Version              string             `json:"version"`
	ThisNode             bool               `json:"thisNode,omitempty"`
}

// A Pool of nodes and buckets.
type Pool struct {
	BucketMap map[string]Bucket
	Nodes     []Node

	BucketURL map[string]string `json:"buckets"`

	client Client
}

// VBucketServerMap is the a mapping of vbuckets to nodes.
type VBucketServerMap struct {
	HashAlgorithm string   `json:"hashAlgorithm"`
	NumReplicas   int      `json:"numReplicas"`
	ServerList    []string `json:"serverList"`
	VBucketMap    [][]int  `json:"vBucketMap"`
}

// Bucket is the primary entry point for most data operations.
type Bucket struct {
	sync.Mutex
	AuthType            string             `json:"authType"`
	Capabilities        []string           `json:"bucketCapabilities"`
	CapabilitiesVersion string             `json:"bucketCapabilitiesVer"`
	Type                string             `json:"bucketType"`
	Name                string             `json:"name"`
	NodeLocator         string             `json:"nodeLocator"`
	Quota               map[string]float64 `json:"quota,omitempty"`
	Replicas            int                `json:"replicaNumber"`
	Password            string             `json:"saslPassword"`
	URI                 string             `json:"uri"`
	StreamingURI        string             `json:"streamingUri"`
	LocalRandomKeyURI   string             `json:"localRandomKeyUri,omitempty"`
	UUID                string             `json:"uuid"`
	DDocs               struct {
		URI string `json:"uri"`
	} `json:"ddocs,omitempty"`
	BasicStats  map[string]interface{} `json:"basicStats,omitempty"`
	Controllers map[string]interface{} `json:"controllers,omitempty"`

	// These are used for JSON IO, but isn't used for processing
	// since it needs to be swapped out safely.
	VBSMJson  VBucketServerMap `json:"vBucketServerMap"`
	NodesJSON []Node           `json:"nodes"`

	pool             *Pool
	connPools        unsafe.Pointer // *[]*connectionPool
	vBucketServerMap unsafe.Pointer // *VBucketServerMap
	nodeList         unsafe.Pointer // *[]Node
	commonSufix      string
	ah               AuthHandler // auth handler
}

// PoolServices is all the bucket-independent services in a pool
type PoolServices struct {
	Rev      int            `json:"rev"`
	NodesExt []NodeServices `json:"nodesExt"`
}

// NodeServices is all the bucket-independent services running on
// a node (given by Hostname)
type NodeServices struct {
	Services map[string]int `json:"services,omitempty"`
	Hostname string         `json:"hostname"`
	ThisNode bool           `json:"thisNode"`
}

type BucketAuth struct {
	name    string
	saslPwd string
	bucket  string
}

func newBucketAuth(name string, pass string, bucket string) *BucketAuth {
	return &BucketAuth{name: name, saslPwd: pass, bucket: bucket}
}

func (ba *BucketAuth) GetCredentials() (string, string, string) {
	return ba.name, ba.saslPwd, ba.bucket
}

// VBServerMap returns the current VBucketServerMap.
func (b *Bucket) VBServerMap() *VBucketServerMap {
	return (*VBucketServerMap)(atomic.LoadPointer(&(b.vBucketServerMap)))
}

func (b *Bucket) GetVBmap(addrs []string) (map[string][]uint16, error) {
	vbmap := b.VBServerMap()
	servers := vbmap.ServerList
	if addrs == nil {
		addrs = vbmap.ServerList
	}

	m := make(map[string][]uint16)
	for _, addr := range addrs {
		m[addr] = make([]uint16, 0)
	}
	for vbno, idxs := range vbmap.VBucketMap {
		addr := servers[idxs[0]]
		if _, ok := m[addr]; ok {
			m[addr] = append(m[addr], uint16(vbno))
		}
	}
	return m, nil
}

// Nodes returns teh current list of nodes servicing this bucket.
func (b Bucket) Nodes() []Node {
	return *(*[]Node)(atomic.LoadPointer(&b.nodeList))
}

// return the list of healthy nodes
func (b Bucket) HealthyNodes() []Node {
	nodes := []Node{}

	for _, n := range b.Nodes() {
		if n.Status == "healthy" && n.CouchAPIBase != "" {
			nodes = append(nodes, n)
		}
		if n.Status != "healthy" { // log non-healthy node
			log.Printf("Non-healthy node; node details:")
			log.Printf("Hostname=%v, Status=%v, CouchAPIBase=%v, ThisNode=%v", n.Hostname, n.Status, n.CouchAPIBase, n.ThisNode)
		}
	}

	return nodes
}

func (b Bucket) getConnPools() []*connectionPool {
	if b.connPools != nil {
		return *(*[]*connectionPool)(atomic.LoadPointer(&b.connPools))
	} else {
		return nil
	}
}

func (b *Bucket) replaceConnPools(with []*connectionPool) {
	for {
		old := atomic.LoadPointer(&b.connPools)
		if atomic.CompareAndSwapPointer(&b.connPools, old, unsafe.Pointer(&with)) {
			if old != nil {
				for _, pool := range *(*[]*connectionPool)(old) {
					if pool != nil {
						pool.Close()
					}
				}
			}
			return
		}
	}
}

func (b Bucket) getConnPool(i int) *connectionPool {

	if i < 0 {
		return nil
	}

	p := b.getConnPools()
	if len(p) > i {
		return p[i]
	}

	return nil
}

func (b Bucket) getConnPoolByHost(host string) *connectionPool {

	pools := b.getConnPools()
	for _, p := range pools {
		if p != nil && p.host == host {
			return p
		}
	}

	return nil
}

// Given a vbucket number, returns a memcached connection to it.
// The connection must be returned to its pool after use.
func (b Bucket) getConnectionToVBucket(vb uint32) (*memcached.Client, *connectionPool, error) {
	for {
		vbm := b.VBServerMap()
		if len(vbm.VBucketMap) < int(vb) {
			return nil, nil, fmt.Errorf("go-couchbase: vbmap smaller than vbucket list: %v vs. %v",
				vb, vbm.VBucketMap)
		}
		masterId := vbm.VBucketMap[vb][0]
		if masterId < 0 {
			return nil, nil, fmt.Errorf("go-couchbase: No master for vbucket %d", vb)
		}
		pool := b.getConnPool(masterId)
		conn, err := pool.Get()
		if err != errClosedPool {
			return conn, pool, err
		}
		// If conn pool was closed, because another goroutine refreshed the vbucket map, retry...
	}
}

func (b Bucket) getMasterNode(i int) string {
	p := b.getConnPools()
	if len(p) > i {
		return p[i].host
	}
	return ""
}

func (b Bucket) authHandler() (ah AuthHandler) {
	if b.pool != nil {
		ah = b.pool.client.ah
	}
	if mbah, ok := ah.(MultiBucketAuthHandler); ok {
		return mbah.ForBucket(b.Name)
	}
	if ah == nil {
		ah = &basicAuth{b.Name, ""}
	}
	return
}

// NodeAddresses gets the (sorted) list of memcached node addresses
// (hostname:port).
func (b Bucket) NodeAddresses() []string {
	vsm := b.VBServerMap()
	rv := make([]string, len(vsm.ServerList))
	copy(rv, vsm.ServerList)
	sort.Strings(rv)
	return rv
}

// CommonAddressSuffix finds the longest common suffix of all
// host:port strings in the node list.
func (b Bucket) CommonAddressSuffix() string {
	input := []string{}
	for _, n := range b.Nodes() {
		input = append(input, n.Hostname)
	}
	return FindCommonSuffix(input)
}

// A Client is the starting point for all services across all buckets
// in a Couchbase cluster.
type Client struct {
	BaseURL *url.URL
	ah      AuthHandler
	Info    Pools
}

func maybeAddAuth(req *http.Request, ah AuthHandler) error {
	if hah, ok := ah.(HTTPAuthHandler); ok {
		return hah.SetCredsForRequest(req)
	}
	if ah != nil {
		user, pass, _ := ah.GetCredentials()
		req.Header.Set("Authorization", "Basic "+
			base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
	}
	return nil
}

// arbitary number, may need to be tuned #FIXME
const HTTP_MAX_RETRY = 5

// Someday golang network packages will implement standard
// error codes. Until then #sigh
func isHttpConnError(err error) bool {

	estr := err.Error()
	return strings.Contains(estr, "broken pipe") ||
		strings.Contains(estr, "broken connection") ||
		strings.Contains(estr, "connection reset")
}

func doHTTPRequest(req *http.Request) (*http.Response, error) {

	var err error
	var res *http.Response

	for i := 0; i < HTTP_MAX_RETRY; i++ {
		res, err = HTTPClient.Do(req)
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

func doPostAPI(
	baseURL *url.URL,
	path string,
	params map[string]interface{},
	authHandler AuthHandler,
	out interface{}) error {

	var requestUrl string

	if q := strings.Index(path, "?"); q > 0 {
		requestUrl = "http://" + baseURL.Host + path[:q] + "?" + path[q+1:]
	} else {
		requestUrl = "http://" + baseURL.Host + path
	}

	postData := url.Values{}
	for k, v := range params {
		postData.Set(k, fmt.Sprintf("%v", v))
	}

	req, err := http.NewRequest("POST", requestUrl, bytes.NewBufferString(postData.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	err = maybeAddAuth(req, authHandler)
	if err != nil {
		return err
	}

	res, err := doHTTPRequest(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		bod, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("HTTP error %v getting %q: %s",
			res.Status, requestUrl, bod)
	}

	d := json.NewDecoder(res.Body)
	if err = d.Decode(&out); err != nil {
		return err
	}
	return nil
}

func queryRestAPI(
	baseURL *url.URL,
	path string,
	authHandler AuthHandler,
	out interface{}) error {

	var requestUrl string

	if q := strings.Index(path, "?"); q > 0 {
		requestUrl = "http://" + baseURL.Host + path[:q] + "?" + path[q+1:]
	} else {
		requestUrl = "http://" + baseURL.Host + path
	}

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return err
	}

	err = maybeAddAuth(req, authHandler)
	if err != nil {
		return err
	}

	res, err := doHTTPRequest(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		bod, _ := ioutil.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("HTTP error %v getting %q: %s",
			res.Status, requestUrl, bod)
	}

	d := json.NewDecoder(res.Body)
	if err = d.Decode(&out); err != nil {
		return err
	}
	return nil
}

func (c *Client) parseURLResponse(path string, out interface{}) error {
	return queryRestAPI(c.BaseURL, path, c.ah, out)
}

func (c *Client) parsePostURLResponse(path string, params map[string]interface{}, out interface{}) error {
	return doPostAPI(c.BaseURL, path, params, c.ah, out)
}

func (b *Bucket) parseURLResponse(path string, out interface{}) error {
	nodes := b.Nodes()
	if len(nodes) == 0 {
		return errors.New("no couch rest URLs")
	}

	// Pick a random node to start querying.
	startNode := rand.Intn(len(nodes))
	maxRetries := len(nodes)
	for i := 0; i < maxRetries; i++ {
		node := nodes[(startNode+i)%len(nodes)] // Wrap around the nodes list.
		// Skip non-healthy nodes.
		if node.Status != "healthy" || node.CouchAPIBase == "" {
			continue
		}
		url := &url.URL{
			Host:   node.Hostname,
			Scheme: "http",
		}

		err := queryRestAPI(url, path, b.pool.client.ah, out)
		if err == nil {
			return err
		}
	}
	return errors.New("all nodes failed to respond")
}

func (b *Bucket) parseAPIResponse(path string, out interface{}) error {
	nodes := b.Nodes()
	if len(nodes) == 0 {
		return errors.New("no couch rest URLs")
	}

	// Pick a random node to start querying.
	startNode := rand.Intn(len(nodes))
	maxRetries := len(nodes)
	for i := 0; i < maxRetries; i++ {
		node := nodes[(startNode+i)%len(nodes)] // Wrap around the nodes list.
		// Skip non-healthy nodes.
		if node.Status != "healthy" || node.CouchAPIBase == "" {
			continue
		}

		u, err := ParseURL(node.CouchAPIBase)
		if err != nil {
			return fmt.Errorf("config error: Bucket %q node #%d CouchAPIBase=%q: %v",
				b.Name, i, node.CouchAPIBase, err)
		} else if b.pool != nil {
			u.User = b.pool.client.BaseURL.User
		}
		u.Path = path

		// generate the path so that the strings are properly escaped
		// MB-13770
		requestPath := strings.Split(u.String(), u.Host)[1]

		err = queryRestAPI(u, requestPath, b.pool.client.ah, out)
		if err == nil {
			return err
		}
	}
	return errors.New("all nodes failed to respond")
}

type basicAuth struct {
	u, p string
}

func (b basicAuth) GetCredentials() (string, string, string) {
	return b.u, b.p, b.u
}

func basicAuthFromURL(us string) (ah AuthHandler) {
	u, err := ParseURL(us)
	if err != nil {
		return
	}
	if user := u.User; user != nil {
		pw, _ := user.Password()
		ah = basicAuth{user.Username(), pw}
	}
	return
}

// ConnectWithAuth connects to a couchbase cluster with the given
// authentication handler.
func ConnectWithAuth(baseU string, ah AuthHandler) (c Client, err error) {
	c.BaseURL, err = ParseURL(baseU)
	if err != nil {
		return
	}
	c.ah = ah

	return c, c.parseURLResponse("/pools", &c.Info)
}

// ConnectWithAuthCreds connects to a couchbase cluster with the give
// authorization creds returned by cb_auth
func ConnectWithAuthCreds(baseU, username, password string) (c Client, err error) {
	c.BaseURL, err = ParseURL(baseU)
	if err != nil {
		return
	}

	c.ah = newBucketAuth(username, password, "")
	return c, c.parseURLResponse("/pools", &c.Info)

}

// Connect to a couchbase cluster.  An authentication handler will be
// created from the userinfo in the URL if provided.
func Connect(baseU string) (Client, error) {
	return ConnectWithAuth(baseU, basicAuthFromURL(baseU))
}

type BucketInfo struct {
	Name     string // name of bucket
	Password string // SASL password of bucket
}

//Get SASL buckets
func GetBucketList(baseU string) (bInfo []BucketInfo, err error) {

	c := &Client{}
	c.BaseURL, err = ParseURL(baseU)
	if err != nil {
		return
	}
	c.ah = basicAuthFromURL(baseU)

	var buckets []Bucket
	err = c.parseURLResponse("/pools/default/buckets", &buckets)
	if err != nil {
		return
	}
	bInfo = make([]BucketInfo, 0)
	for _, bucket := range buckets {
		bucketInfo := BucketInfo{Name: bucket.Name, Password: bucket.Password}
		bInfo = append(bInfo, bucketInfo)
	}
	return bInfo, err
}

//Set viewUpdateDaemonOptions
func SetViewUpdateParams(baseU string, params map[string]interface{}) (viewOpts map[string]interface{}, err error) {

	c := &Client{}
	c.BaseURL, err = ParseURL(baseU)
	if err != nil {
		return
	}
	c.ah = basicAuthFromURL(baseU)

	if len(params) < 1 {
		return nil, fmt.Errorf("No params to set")
	}

	err = c.parsePostURLResponse("/settings/viewUpdateDaemon", params, &viewOpts)
	if err != nil {
		return
	}
	return viewOpts, err
}

func (b *Bucket) Refresh() error {
	pool := b.pool
	tmpb := &Bucket{}
	err := pool.client.parseURLResponse(b.URI, tmpb)
	if err != nil {
		return err
	}

	pools := b.getConnPools()

	// We need this lock to ensure that bucket refreshes happening because
	// of NMVb errors received during bulkGet do not end up over-writing
	// pool.inUse.
	b.Lock()

	for _, pool := range pools {
		if pool != nil {
			pool.inUse = false
		}
	}

	newcps := make([]*connectionPool, len(tmpb.VBSMJson.ServerList))
	for i := range newcps {

		pool := b.getConnPoolByHost(tmpb.VBSMJson.ServerList[i])
		if pool != nil && pool.inUse == false {
			// if the hostname and index is unchanged then reuse this pool
			newcps[i] = pool
			pool.inUse = true
			continue
		}

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
	return nil
}

func (p *Pool) refresh() (err error) {
	p.BucketMap = make(map[string]Bucket)

	buckets := []Bucket{}
	err = p.client.parseURLResponse(p.BucketURL["uri"], &buckets)
	if err != nil {
		return err
	}
	for _, b := range buckets {
		b.pool = p
		b.nodeList = unsafe.Pointer(&b.NodesJSON)
		b.replaceConnPools(make([]*connectionPool, len(b.VBSMJson.ServerList)))

		p.BucketMap[b.Name] = b
	}
	return nil
}

// GetPool gets a pool from within the couchbase cluster (usually
// "default").
func (c *Client) GetPool(name string) (p Pool, err error) {
	var poolURI string
	for _, p := range c.Info.Pools {
		if p.Name == name {
			poolURI = p.URI
		}
	}
	if poolURI == "" {
		return p, errors.New("No pool named " + name)
	}

	err = c.parseURLResponse(poolURI, &p)

	p.client = *c

	err = p.refresh()
	return
}

// GetPoolServices returns all the bucket-independent services in a pool.
// (See "Exposing services outside of bucket context" in http://goo.gl/uuXRkV)
func (c *Client) GetPoolServices(name string) (ps PoolServices, err error) {
	var poolName string
	for _, p := range c.Info.Pools {
		if p.Name == name {
			poolName = p.Name
		}
	}
	if poolName == "" {
		return ps, errors.New("No pool named " + name)
	}

	poolURI := "/pools/" + poolName + "/nodeServices"
	err = c.parseURLResponse(poolURI, &ps)

	return
}

// Close marks this bucket as no longer needed, closing connections it
// may have open.
func (b *Bucket) Close() {
	if b.connPools != nil {
		for _, c := range b.getConnPools() {
			if c != nil {
				c.Close()
			}
		}
		b.connPools = nil
	}
}

func bucketFinalizer(b *Bucket) {
	if b.connPools != nil {
		log.Printf("Warning: Finalizing a bucket with active connections.")
	}
}

// GetBucket gets a bucket from within this pool.
func (p *Pool) GetBucket(name string) (*Bucket, error) {
	rv, ok := p.BucketMap[name]
	if !ok {
		return nil, errors.New("No bucket named " + name)
	}
	runtime.SetFinalizer(&rv, bucketFinalizer)
	err := rv.Refresh()
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// GetBucket gets a bucket from within this pool.
func (p *Pool) GetBucketWithAuth(bucket, username, password string) (*Bucket, error) {
	rv, ok := p.BucketMap[bucket]
	if !ok {
		return nil, errors.New("No bucket named " + bucket)
	}
	runtime.SetFinalizer(&rv, bucketFinalizer)
	rv.ah = newBucketAuth(username, password, bucket)
	err := rv.Refresh()
	if err != nil {
		return nil, err
	}
	return &rv, nil
}

// GetPool gets the pool to which this bucket belongs.
func (b *Bucket) GetPool() *Pool {
	return b.pool
}

// GetClient gets the client from which we got this pool.
func (p *Pool) GetClient() *Client {
	return &p.client
}

// GetBucket is a convenience function for getting a named bucket from
// a URL
func GetBucket(endpoint, poolname, bucketname string) (*Bucket, error) {
	var err error
	client, err := Connect(endpoint)
	if err != nil {
		return nil, err
	}

	pool, err := client.GetPool(poolname)
	if err != nil {
		return nil, err
	}

	return pool.GetBucket(bucketname)
}
