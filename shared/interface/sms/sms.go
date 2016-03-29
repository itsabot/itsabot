// Package sms enables interaction with arbitrary SMS providers. It implements a
// standardized interface through which Twilio, Nexmo and more may be supported.
// It's up to individual drivers to add support for each of these services.
package sms

import (
	"fmt"
	"sort"
	"sync"

	"github.com/itsabot/abot/shared/interface/sms/driver"
	"github.com/julienschmidt/httprouter"
)

var driversMu sync.RWMutex
var drivers = make(map[string]driver.Driver)

// Register makes a calendar driver available by the provided name. If Register
// is called twice with the same name or if driver is nill, it panics.
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("cal: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("cal: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// Conn is a connection to a specific sms driver.
type Conn struct {
	driver driver.Driver
	conn   driver.Conn
}

// Open a connection to a registered driver.
func Open(driverName string, r *httprouter.Router) (*Conn, error) {
	driversMu.RLock()
	driveri, ok := drivers[driverName]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sms: unknown driver %q (forgotten import?)",
			driverName)
	}
	conn, err := driveri.Open(r)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		driver: driveri,
		conn:   conn,
	}
	return c, nil
}

// Driver returns the driver used by a connection.
func (c *Conn) Driver() driver.Driver {
	return c.driver
}

// Send an SMS message through an opened driver connection. The from number is
// handled by the driver.
func (c *Conn) Send(to, msg string) error {
	return c.conn.Send(to, msg)
}
