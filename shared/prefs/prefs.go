// Package prefs manages global preferences that plugins frequently share with
// one another.
package prefs

// These are the const keys used in memory to store frequently accessed data.
const (
	Name            = "name"
	Location        = "location"
	HomeAddress     = "home_address"
	ShippingAddress = "shipping_address"
	WorkAddress     = "work_address"
)
