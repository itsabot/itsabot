package main

import "testing"

var test *testing.T

/*
func findStructuredObjsTest(t *testing.T) {
	dict, err := loadDictionary()
	if err != nil {
		log.Fatalln("failed loading dict", err)
	}
	testData := map[string][]StructuredInput{
		"Stop talking.": []StructuredInput{
			StructuredInput{
				Command: "Stop",
				Objects: []string{"talking"},
			},
		},
		"Complete this assignment.": []StructuredInput{
			StructuredInput{
				Command: "Complete",
				Objects: []string{"assignment"},
			},
		},
		"Order me an Uber.": []StructuredInput{
			StructuredInput{
				Command: "Order",
				Actors:  []string{"me"},
				Objects: []string{"Uber"},
			},
		},
		"Buy 2 pepperoni pizzas for the office, and bill it to the office.": []StructuredInput{
			StructuredInput{
				Command: "Buy",
				Actors:  []string{"office"},
				Objects: []string{"2 pepperoni pizzas"},
			},
			StructuredInput{
				Command:  "bill",
				Actors:   []string{"it"},
				Objects:  []string{"office"},
				Contexts: []string{"Order"},
			},
		},
	}
	for _, s := range testData {
		testEq(findStructuredObjs(s))
	}
}
*/
