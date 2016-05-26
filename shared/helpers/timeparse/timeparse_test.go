package timeparse

import (
	"os"
	"testing"
	"time"

	"github.com/itsabot/abot/core/log"
)

func TestMain(m *testing.M) {
	log.SetDebug(true)
	if err := os.Setenv("ABOT_DEBUG", "true"); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestParse(t *testing.T) {
	n := time.Now()
	// _, zone := n.Zone()
	n.Add(-6 * time.Hour)
	tests := map[string][]time.Time{
		"2pm":                []time.Time{time.Date(n.Year(), n.Month(), n.Day(), 14, 0, 0, 0, n.Location())},
		"2 am":               []time.Time{time.Date(n.Year(), n.Month(), n.Day(), 2, 0, 0, 0, n.Location())},
		"at 2 p.m.":          []time.Time{time.Date(n.Year(), n.Month(), n.Day(), 14, 0, 0, 0, n.Location())},
		"2pm tomorrow":       []time.Time{time.Date(n.Year(), n.Month(), n.Day()+1, 14, 0, 0, 0, n.Location())},
		"2am yesterday":      []time.Time{time.Date(n.Year(), n.Month(), n.Day()-1, 2, 0, 0, 0, n.Location())},
		"2 days ago":         []time.Time{time.Date(n.Year(), n.Month(), n.Day()-2, 9, 0, 0, 0, n.Location())},
		"in 3 days from now": []time.Time{time.Date(n.Year(), n.Month(), n.Day()+3, 9, 0, 0, 0, n.Location())},
		"1 week":             []time.Time{time.Date(n.Year(), n.Month(), n.Day()+7, 9, 0, 0, 0, n.Location())},
		"1 week ago":         []time.Time{time.Date(n.Year(), n.Month(), n.Day()-7, 9, 0, 0, 0, n.Location())},
		"in a year":          []time.Time{time.Date(n.Year()+1, n.Month(), n.Day(), 9, 0, 0, 0, n.Location())},
		"next year":          []time.Time{time.Date(n.Year()+1, n.Month(), n.Day(), 9, 0, 0, 0, n.Location())},
		"in 4 weeks":         []time.Time{time.Date(n.Year(), n.Month(), n.Day()+28, 9, 0, 0, 0, n.Location())},
		"later today":        []time.Time{time.Date(n.Year(), n.Month(), n.Day(), n.Hour()+6, n.Minute(), 0, 0, n.Location())},
		"a few hours":        []time.Time{time.Date(n.Year(), n.Month(), n.Day(), n.Hour()+2, n.Minute(), 0, 0, n.Location())},
		"in 30 mins":         []time.Time{time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute()+30, 0, 0, n.Location())},
		"in 2 hours":         []time.Time{time.Date(n.Year(), n.Month(), n.Day(), n.Hour()+2, n.Minute(), 0, 0, n.Location())},
		"invalid time":       []time.Time{},
		"May 2050":           []time.Time{time.Date(2050, 5, 1, 9, 0, 0, 0, n.Location())},
		"June 26 2050":       []time.Time{time.Date(2050, 6, 26, 0, 0, 0, 0, n.Location())},
		"June 26th 2050":     []time.Time{time.Date(2050, 6, 26, 0, 0, 0, 0, n.Location())},
		"at 2 tomorrow": []time.Time{
			time.Date(n.Year(), n.Month(), n.Day()+1, 2, 0, 0, 0, n.Location()),
			time.Date(n.Year(), n.Month(), n.Day()+1, 14, 0, 0, 0, n.Location()),
		},
		"now": []time.Time{time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), n.Location())},
		/*
			"2 days ago at 6PM":  []time.Time{time.Date(n.Year(), n.Month(), n.Day()-2, 18, 0, 0, 0, n.Location())},
			"12PM EST":           []time.Time{time.Date(n.Year(), n.Month(), n.Day(), 12-zone, n.Minute(), 0, 0, n.Location())},
		*/
	}
	for test, exp := range tests {
		log.Debug("test:", test)
		res := Parse(test)
		if len(res) == 0 {
			if len(exp) == 0 {
				continue
			}
			t.Fatalf("expected %q, got none", exp)
		}
		if len(exp) == 0 && len(res) > 0 {
			t.Fatalf("expected none, but got %q", res)
		}
		if !exp[0].Equal(res[0]) && exp[0].Sub(res[0]) > 2*time.Minute {
			t.Fatalf("expected %q, got %q", exp, res)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	log.SetDebug(false)
	for i := 0; i < b.N; i++ {
		_ = Parse("2 p.m. tomorrow")
	}
}
