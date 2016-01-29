package main

import (
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/avabot/ava/shared/datatypes"
)

// TODO customize the type of m.State, forcing all reads and writes through
// these getter/setter functions to preserve and handle types across interface{}
func getOffset() uint {
	switch m.State["offset"].(type) {
	case uint:
		return m.State["offset"].(uint)
	case int:
		return uint(m.State["offset"].(int))
	case float64:
		return uint(m.State["offset"].(float64))
	default:
		l.WithField("type", reflect.TypeOf(m.State["offset"])).
			Errorln("couldn't get offset: invalid type")
	}
	return uint(0)
}

func getQuery() string {
	return m.State["query"].(string)
}

func getCategory() string {
	return m.State["category"].(string)
}

func getBudget() uint64 {
	switch m.State["budget"].(type) {
	case int64:
		return uint64(m.State["budget"].(int64))
	case uint64:
		return m.State["budget"].(uint64)
	case float64:
		return uint64(m.State["budget"].(float64))
	case string:
		s, err := strconv.ParseUint(m.State["budget"].(string), 10,
			64)
		if err != nil {
			l.WithField("budget", s).Errorln(
				"couldn't get budget: convert from string")
		}
		return s
	default:
		l.WithField("type", reflect.TypeOf(m.State["budget"])).
			Errorln("couldn't get budget: invalid type")
	}
	return uint64(0)
}

func getShippingAddress() *dt.Address {
	addr, ok := m.State["shippingAddress"].(*dt.Address)
	if !ok {
		return nil
	}
	return addr
}

func getSelectedProducts() dt.ProductSels {
	products, ok := m.State["productsSelected"].([]dt.ProductSel)
	if !ok {
		prodMap, ok := m.State["productsSelected"].(interface{})
		if !ok {
			l.Errorln("productsSelected not found",
				m.State["productsSelected"])
			return nil
		}
		byt, err := json.Marshal(prodMap)
		if err != nil {
			l.Errorln("marshaling products", err)
		}
		if err = json.Unmarshal(byt, &products); err != nil {
			l.Errorln("unmarshaling products", err)
		}
	}
	return products
}

func getRecommendations() []dt.Product {
	products, ok := m.State["recommendations"].([]dt.Product)
	if !ok {
		prodMap, ok := m.State["recommendations"].(interface{})
		if !ok {
			l.Errorln("recommendations not found",
				m.State["recommendations"])
			return nil
		}
		byt, err := json.Marshal(prodMap)
		if err != nil {
			l.Errorln("marshaling products", err)
		}
		if err = json.Unmarshal(byt, &products); err != nil {
			l.Errorln("unmarshaling products", err)
		}
		return nil
	}
	return products
}

func getState() float64 {
	state, ok := m.State["state"].(float64)
	if !ok {
		state = 0.0
	}
	return state
}

func setOffset(state float64) {
	m.State["offset"] = state
}

func setQuery(q string) {
	m.State["query"] = q
}

func setState(state float64) {
	m.State["state"] = state
}

func setCategory(cat string) {
	m.State["category"] = cat
}

func setBudget(budg uint64) {
	m.State["budget"] = budg
}
