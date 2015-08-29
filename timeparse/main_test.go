package main

import (
	"log"
	"testing"
	"time"
)

var test *testing.T

func TestParse(t *testing.T) {
	test = t
	loc := locations()

	// NOTE: Function assumes a current date of Friday, July 31, 2015 in
	// Los Angeles, CA.
	testDate := time.Date(2015, time.July, 31, 9, 0, 0, 0, loc["pacific"])

	testData := map[string][]time.Time{
		// Test basic times
		"2": []time.Time{
			time.Date(2015, time.July, 31, 2, 0, 0, 0, loc["pacific"]),
			time.Date(2015, time.July, 31, 14, 0, 0, 0, loc["pacific"]),
		},
		"2AM": []time.Time{
			time.Date(2015, time.July, 31, 2, 0, 0, 0, loc["pacific"]),
		},
		"2 A.M.": []time.Time{
			time.Date(2015, time.July, 31, 2, 0, 0, 0, loc["pacific"]),
		},
		"2 P.M": []time.Time{
			time.Date(2015, time.July, 31, 14, 0, 0, 0, loc["pacific"]),
		},
		"2:30pm": []time.Time{
			time.Date(2015, time.July, 31, 14, 30, 0, 0, loc["pacific"]),
		},
		/*
			"1 - 2 A.M.": []time.Time{
				time.Date(2015, time.August, 4, 1, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 2, 0, 0, 0, loc["pacific"]),
			},
			"1-2AM or 6-8PM": []time.Time{
				time.Date(2015, time.August, 4, 1, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 2, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 18, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 19, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 20, 0, 0, 0, loc["pacific"]),
			},
			"11AM-2PM": []time.Time{
				time.Date(2015, time.August, 4, 11, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 12, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 13, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 14, 0, 0, 0, loc["pacific"]),
			},
		*/

		// TODO: Test minutes, e.g. 9:30-11AM
		// TODO: Test military time (e.g. 0600), 24-hour time formats
		// TODO: Test natural language: Today, Tomorrow, Yesterday, etc.

		// Test timezone detection.
		// NOTE: Golang handles daylight savings.
		"Tuesday at 6 Pacific": []time.Time{
			time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["pacific"]),
			time.Date(2015, time.August, 4, 18, 0, 0, 0, loc["pacific"]),
		},
		"Tuesday at 6AM Mountain": []time.Time{
			time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["mountain"]),
		},
		"Tuesday at 6AM ET": []time.Time{
			time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["eastern"]),
		},
		"Tuesday at 6AM CEST": []time.Time{
			time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["gmt"]),
		},
		"Tuesday at 6AM UTC": []time.Time{
			time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["utc"]),
		},
		/*

			// Test assumed timezones
			"6AM UTC or 7AM": []time.Time{
				time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["utc"]),
				time.Date(2015, time.August, 4, 7, 0, 0, 0, loc["utc"]),
			},
			"6AM or 7AM UTC": []time.Time{
				time.Date(2015, time.August, 4, 6, 0, 0, 0, loc["utc"]),
				time.Date(2015, time.August, 4, 7, 0, 0, 0, loc["utc"]),
			},

			// Test relative dates
			"This Tuesday": []time.Time{
				time.Date(2015, time.July, 28, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
			},
			"Tuesday after next": []time.Time{
				time.Date(2015, time.August, 11, 0, 0, 0, 0, loc["pacific"]),
			},
			"Last Tuesday": []time.Time{
				time.Date(2015, time.July, 28, 0, 0, 0, 0, loc["pacific"]),
			},
			"Tuesday before last": []time.Time{
				time.Date(2015, time.July, 21, 0, 0, 0, 0, loc["pacific"]),
			},
			"Next Tuesday": []time.Time{
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
			},
			"Friday": []time.Time{
				time.Date(2015, time.July, 31, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 7, 0, 0, 0, 0, loc["pacific"]),
			},
			"Next Friday": []time.Time{
				time.Date(2015, time.August, 7, 0, 0, 0, 0, loc["pacific"]),
			},
			"Last Friday": []time.Time{
				time.Date(2015, time.July, 24, 0, 0, 0, 0, loc["pacific"]),
			},
			"Thursday": []time.Time{
				time.Date(2015, time.July, 30, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 6, 0, 0, 0, 0, loc["pacific"]),
			},
			"Last Thursday": []time.Time{
				time.Date(2015, time.July, 30, 0, 0, 0, 0, loc["pacific"]),
			},
			"February": []time.Time{
				time.Date(2015, time.February, 1, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2016, time.February, 1, 0, 0, 0, 0, loc["pacific"]),
			},
			"Next February": []time.Time{
				time.Date(2016, time.February, 1, 0, 0, 0, 0, loc["pacific"]),
			},
			"Next year": []time.Time{
				time.Date(2016, time.January, 1, 0, 0, 0, 0, loc["pacific"]),
			},

			// Test dates
			"Feb. 20th": []time.Time{
				time.Date(2015, time.February, 20, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2016, time.February, 20, 0, 0, 0, 0, loc["pacific"]),
			},
			"February 20": []time.Time{
				time.Date(2015, time.February, 20, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2016, time.February, 20, 0, 0, 0, 0, loc["pacific"]),
			},
			"February 20, 2019": []time.Time{
				time.Date(2019, time.February, 20, 0, 0, 0, 0, loc["pacific"]),
			},
			"7/31": []time.Time{
				time.Date(2015, time.July, 31, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2016, time.July, 31, 0, 0, 0, 0, loc["pacific"]),
			},
			"Next 7/31": []time.Time{
				time.Date(2016, time.July, 31, 0, 0, 0, 0, loc["pacific"]),
			},
			"Last 7/31": []time.Time{
				time.Date(2014, time.July, 31, 0, 0, 0, 0, loc["pacific"]),
			},
			"This 7/31": []time.Time{
				time.Date(2015, time.July, 31, 0, 0, 0, 0, loc["pacific"]),
			},

			// Test multiple dates and times
			"Tuesday or Wednesday": []time.Time{
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 5, 0, 0, 0, 0, loc["pacific"]),
			},
			"Tuesday and Wednesday": []time.Time{
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 5, 0, 0, 0, 0, loc["pacific"]),
			},
			"Tuesday through Thursday": []time.Time{
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 5, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 6, 0, 0, 0, 0, loc["pacific"]),
			},
			"Tues-Thurs": []time.Time{
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 5, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 6, 0, 0, 0, 0, loc["pacific"]),
			},
			"Tues to Thurs": []time.Time{
				time.Date(2015, time.August, 4, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 5, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 6, 0, 0, 0, 0, loc["pacific"]),
			},

			// NOTE: Follows American convention that Sunday is the first
			// day of the week.
			"Next week": []time.Time{
				time.Date(2015, time.August, 2, 0, 0, 0, 0, loc["pacific"]),
			},
			"This week": []time.Time{
				time.Date(2015, time.July, 27, 0, 0, 0, 0, loc["pacific"]),
			},
			"Last week": []time.Time{
				time.Date(2015, time.July, 19, 0, 0, 0, 0, loc["pacific"]),
			},

			// Test more complex expressions
			// NOTE: "This Wednesday" here isn't deterministic. It could
			// either refer to the most recent (past) Wednesday, or the
			// next one. You'll have to look at the sentence or program
			// context to determine what's meant.
			"Tuesday at 6PM": []time.Time{
				time.Date(2015, time.August, 4, 18, 0, 0, 0, loc["pacific"]),
			},
			"Going to the store this Wednesday at 9": []time.Time{
				time.Date(2015, time.August, 30, 0, 0, 0, 0, loc["pacific"]),
				time.Date(2015, time.August, 30, 0, 0, 0, 0, loc["pacific"]),
			},
		*/
	}

	for expression, expectedTimes := range testData {
		times, err := ParseFromTime(testDate, expression)
		check(err)
		if !testEq(times, expectedTimes) {
			log.Println("Expected", expectedTimes)
			log.Println("Received", times)
			t.Fail()
		}
	}
}

func locations() map[string]*time.Location {
	loc := map[string]*time.Location{}
	var err error
	loc["pacific"], err = time.LoadLocation("America/Los_Angeles")
	check(err)
	loc["mountain"], err = time.LoadLocation("America/Denver")
	check(err)
	loc["eastern"], err = time.LoadLocation("America/New_York")
	check(err)
	loc["gmt"], err = time.LoadLocation("Europe/Zurich")
	check(err)
	loc["utc"], err = time.LoadLocation("UTC")
	check(err)
	return loc
}

func check(err error) {
	if err != nil {
		test.Fatal(err)
	}
}

func testEq(a, b []time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
