package dt

import (
	"math"
)

// Product represents a product result returned from ElasticSearch. Note that
// because it's an ElasticSearch result, it has a string ID.
type Product struct {
	ID        string
	Name      string
	Size      string
	Stock     uint
	Price     uint64
	VendorID  uint64
	Category  string
	Varietals []string
	Reviews   []struct {
		Score uint
		Body  string
	}
}

// ProductSel is a user's product selection, keeping track of both the product
// selected and the quantity desired.
type ProductSel struct {
	*Product
	Count uint
}

// ProductSels represents a slice of product selections, adding a helper method
// that makes it easy to calculate the prices (subtotal, tax, shipping, and
// total).
type ProductSels []ProductSel

// ProductSels represents a slice of product selections, adding a helper method
// that makes it easy to calculate the prices (subtotal, tax, shipping, and
// total) for a given group of product selections.
func (prods ProductSels) Prices(addr *Address) map[string]uint64 {
	m := map[string]uint64{
		"products": 0,
		"tax":      0,
		"shipping": 0,
		"total":    0,
	}
	for _, prod := range prods {
		m["products"] += prod.Price * uint64(prod.Count)
	}
	// TODO
	// Calculate shipping. Note that this is vendor specific, so this should
	// be moved to the Vendors table in the database.
	m["shipping"] = 1290 + uint64((len(prods)-1)*120)
	var tax float64
	if addr != nil {
		tax = statesTax[addr.State]
		if tax > 0.0 {
			tax *= float64(m["products"])
		}
	}
	m["tax"] = uint64(math.Ceil(tax))
	m["total"] = m["products"] + m["shipping"] + m["tax"]
	return m
}
