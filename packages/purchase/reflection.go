package main

import (
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/avabot/ava/shared/datatypes"
)

// TODO customize the type of resp.State, forcing all reads and writes through
// these getter/setter functions to preserve and handle types across interface{}
func getOffset() uint {
	switch resp.State["offset"].(type) {
	case uint:
		return resp.State["offset"].(uint)
	case int:
		return uint(resp.State["offset"].(int))
	case float64:
		return uint(resp.State["offset"].(float64))
	default:
		l.WithField("type", reflect.TypeOf(resp.State["offset"])).
			Errorln("couldn't get offset: invalid type")
	}
	return uint(0)
}

func getQuery() string {
	return resp.State["query"].(string)
}

func getCategory() string {
	return resp.State["category"].(string)
}

func getBudget() uint64 {
	switch resp.State["budget"].(type) {
	case int64:
		return uint64(resp.State["budget"].(int64))
	case uint64:
		return resp.State["budget"].(uint64)
	case float64:
		return uint64(resp.State["budget"].(float64))
	case string:
		s, err := strconv.ParseUint(resp.State["budget"].(string), 10,
			64)
		if err != nil {
			l.WithField("budget", s).Errorln(
				"couldn't get budget: convert from string")
		}
		return s
	default:
		l.WithField("type", reflect.TypeOf(resp.State["budget"])).
			Errorln("couldn't get budget: invalid type")
	}
	return uint64(0)
}

func getShippingAddress() *dt.Address {
	addr, ok := resp.State["shippingAddress"].(*dt.Address)
	if !ok {
		return nil
	}
	return addr
}

func getSelectedProducts() dt.ProductSels {
	products, ok := resp.State["productsSelected"].([]dt.ProductSel)
	if !ok {
		prodMap, ok := resp.State["productsSelected"].(interface{})
		if !ok {
			l.Errorln("productsSelected not found",
				resp.State["productsSelected"])
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
	products, ok := resp.State["recommendations"].([]dt.Product)
	if !ok {
		prodMap, ok := resp.State["recommendations"].(interface{})
		if !ok {
			l.Errorln("recommendations not found",
				resp.State["recommendations"])
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
	state, ok := resp.State["state"].(float64)
	if !ok {
		state = 0.0
	}
	return state
}

func setOffset(state float64) {
	resp.State["offset"] = state
}

func setQuery(q string) {
	resp.State["query"] = q
}

func setState(state float64) {
	resp.State["state"] = state
}

func setCategory(cat string) {
	resp.State["category"] = cat
}

func setBudget(budg uint64) {
	resp.State["budget"] = budg
}
