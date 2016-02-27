// Package payment enables interaction with arbitrary payment providers. It
// implements a standardized interface through which Stripe, PayPal and more may
// be supported. It's up to individual drivers to add support for each of these
// services.
package payment

import (
	"database/sql/driver"
	"sort"
	"sync"
)

var driversMu sync.RWMutex
var drivers = make(map[string]driver.Driver)

// Register makes a payment driver available by the provided name. If Register
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
