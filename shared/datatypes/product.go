package dt

import (
	"math"
)

// Product represents a product result returned from ElasticSearch. Note that
// because it's an ElasticSearch result, it has a string ID.
type Product struct {
	ID       string
	Name     string
	Size     string
	Stock    uint
	Price    uint64
	VendorID uint64
	Reviews  []struct {
		Score uint
		Body  string
	}
}

type ProductSel struct {
	*Product
	Count uint
}

type ProductSels []ProductSel

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
	// calculate shipping. note that this is vendor specific
	m["shipping"] = 1290 + uint64((len(prods)-1)*120)
	tax := statesTax[addr.State]
	if tax > 0.0 {
		tax *= float64(m["total"])
	}
	m["tax"] = uint64(math.Ceil(tax))
	m["total"] = m["product"] + m["shipping"] + m["tax"]
	return m
}
