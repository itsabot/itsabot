// Package sms enables interaction with arbitrary SMS providers. It implements a
// standardized interface through which Twilio, Nexmo and more may be supported.
// It's up to individual drivers to add support for each of these services.
package sms

import (
	"fmt"
	"sort"
	"sync"

	"github.com/itsabot/abot/shared/interface/sms/driver"
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

type Conn struct {
	driver driver.Driver
	conn   driver.Conn
}

func Open(driverName, auth string) (*Conn, error) {
	driversMu.RLock()
	driveri, ok := drivers[driverName]
	driversMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("sms: unknown driver %q (forgotten import?)",
			driverName)
	}
	conn, err := driveri.Open(auth)
	if err != nil {
		return nil, err
	}
	c := &Conn{
		driver: driveri,
		conn:   conn,
	}
	return c, nil
}

func (c *Conn) Send(from, to, msg string) error {
	return c.conn.Send(from, to, msg)
}

func (c *Conn) Driver() driver.Driver {
	return c.driver
}
