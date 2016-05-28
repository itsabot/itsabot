// Package timeparse parses times in strings in a wide variety of formats to
// corresponding time.Times.
package timeparse

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/itsabot/abot/core/log"
)

// timeLocation tracks the timezone of a time. Since time.Location.String()
// defaults to "UTC", the bool tz.utc tracks if the timezone was deliberately
// set to UTC.
type timeLocation struct {
	loc *time.Location
	utc bool
}

// timeTransform is an enum type.
type timeTransform int

// These transform keys allow us to track the type of transform we're making to
// a time. Use the closest transform you can to the size without going over,
// e.g. "Next week" would use transform transformDay.
const (
	transformInvalid timeTransform = iota
	transformYear
	transformMonth
	transformDay
	transformHour
	transformMinute
)

// TimeContext is a type used to extrapolate a single piece of info, like AMPM
// or the month across multiple times. If any item in TimeContext is nil, the
// information returned in the []time.Time slice corresponding to that was
// arrived at on a best-guess basis. You can (and should) add a logic layer to
// your own application which selects the most appropriate time based on your
// known context beyond the text provided to timeparse.
type TimeContext struct {
	year  int
	month time.Month
	day   int
	tz    timeLocation
	ampm  int
}

// ErrInvalidTimeFormat is returned when time.Parse could not parse the time
// across all known valid time formats. For a list of valid time formats, see
// the unexported variables timeFormatsNoDay and timeFormatsWithDay in
// shared/helpers/timeparse.
var ErrInvalidTimeFormat = errors.New("invalid time format")

const (
	noDay  = false
	hasDay = true
)

const (
	ampmNoTime = iota
	amTime
	pmTime
)

var timeFormats = []string{
	"15PM",
	"15:4",
	"15:4PM",
	"15:4:5",
	"15:4:5PM",
	"Mon 15",
	"Mon 15PM",
	"Mon 15 MST",
	"Mon 15PM MST",
	"Mon 15:4 MST",
	"Mon 15:4PM MST",
	"Mon 15:4:5 MST",
	"Mon 15:4:5PM MST",
	"15:4:5 Jan 2 06",
	"15:4:5PM Jan 2 06",
	"15:4:5 Jan 2 2006",
	"15:4:5PM Jan 2 2006",
	"15:4:5 Jan 2 2006 MST",
	"15:4:5PM Jan 2 2006 MST",
	"Jan 2",
	"Jan 2 15:4PM",
	"Mon Jan 2",
	"Mon Jan 2 06",
	"Mon Jan 2 2006",
	"Jan 2 2006",
	"Jan 2006",
	"1/2/06",
	"1/2/2006",
	"1/2/06 15:4:5",
	"1/2/06 15:4:5PM",
	"1/2/2006 15:4:5",
	"1/2/2006 15:4:5PM",
	"2006-1-2 15:4:5",
	"2006-1-2 15:4",
	"2006-1-2",
	"1-2",
}

// Parse a natural language string to determine most likely times based on the
// current system time.
func Parse(nlTime string) []time.Time {
	return ParseFromTime(time.Now(), nlTime)
}

// ParseFromTime parses a natural language string to determine most likely times
// based on a set time "context." The time context changes the meaning of words
// like "this Tuesday," "next Tuesday," etc.
func ParseFromTime(t time.Time, nlTime string) []time.Time {
	r := strings.NewReplacer(
		".", "",
		",", "",
		"(", "",
		")", "",
		"'", "",
	)
	nlTime = r.Replace(nlTime)
	nlTime = strings.ToLower(nlTime)
	r = strings.NewReplacer(
		"at ", "",
		"time", "",
		"oclock", "",
		"am", "AM",
		"pm", "PM",
		" am", "AM",
		" pm", "PM",

		// 1st, 2nd, 3rd, 4th, 5th, etc.
		"1st", "1",
		"2nd", "2",
		"3rd", "3",
		"th", "",
		"21st", "21",
		"22nd", "22",
		"23rd", "23",
		"31st", "31",
	)
	nlTime = r.Replace(nlTime)
	nlTime = strings.Title(nlTime)
	if nlTime == "Now" {
		return []time.Time{time.Now()}
	}
	st := strings.Fields(nlTime)
	transform := struct {
		Transform  int
		Type       timeTransform
		Multiplier int
	}{
		Multiplier: 1,
	}
	stFull := ""
	var closeTime bool
	var idxRel int
	var loc *time.Location
	for i := range st {
		// Normalize days
		switch st[i] {
		case "Monday":
			st[i] = "Mon"
			transform.Type = transformDay
		case "Tuesday", "Tues":
			st[i] = "Tue"
			transform.Type = transformDay
		case "Wednesday":
			st[i] = "Wed"
			transform.Type = transformDay
		case "Thursday", "Thur", "Thurs":
			st[i] = "Thu"
			transform.Type = transformDay
		case "Friday":
			st[i] = "Fri"
			transform.Type = transformDay
		case "Saturday":
			st[i] = "Sat"
			transform.Type = transformDay
		case "Sunday":
			st[i] = "Sun"
			transform.Type = transformDay

		// Normalize months
		case "January":
			st[i] = "Jan"
			transform.Type = transformMonth
		case "February":
			st[i] = "Feb"
			transform.Type = transformMonth
		case "March":
			st[i] = "Mar"
			transform.Type = transformMonth
		case "April":
			st[i] = "Apr"
			transform.Type = transformMonth
		case "May":
			// No translation needed for May
			transform.Type = transformMonth
		case "June":
			st[i] = "Jun"
			transform.Type = transformMonth
		case "July":
			st[i] = "Jul"
			transform.Type = transformMonth
		case "August":
			st[i] = "Aug"
			transform.Type = transformMonth
		case "September", "Sept":
			st[i] = "Sep"
			transform.Type = transformMonth
		case "October":
			st[i] = "Oct"
			transform.Type = transformMonth
		case "November":
			st[i] = "Nov"
			transform.Type = transformMonth
		case "December":
			st[i] = "Dec"
			transform.Type = transformMonth

		// If non-deterministic timezone information is provided,
		// e.g. ET or Eastern rather than EST, then load the location.
		// Daylight Savings will be determined on parsing
		case "Pacific", "PT":
			st[i] = ""
			loc = loadLocation("America/Los_Angeles")
		case "Mountain", "MT":
			st[i] = ""
			loc = loadLocation("America/Denver")
		case "Central", "CT":
			st[i] = ""
			loc = loadLocation("America/Chicago")
		case "Eastern", "ET":
			st[i] = ""
			loc = loadLocation("America/New_York")
		// TODO Add the remaining timezones

		// Handle relative times. This currently does not handle
		// complex cases like "in 3 months and 2 days"
		case "Yesterday":
			st[i] = ""
			transform.Type = transformDay
			transform.Transform = 1
			transform.Multiplier = -1
		case "Tomorrow":
			st[i] = ""
			transform.Transform = 1
			transform.Type = transformDay
		case "Today":
			st[i] = ""
			closeTime = true
			transform.Type = transformHour
		case "Ago", "Last":
			st[i] = ""
			transform.Transform = 1
			transform.Multiplier *= -1
		// e.g. "In an hour"
		case "Next", "From", "Now", "In":
			st[i] = ""
			transform.Transform = 1
		case "Later":
			st[i] = ""
			transform.Transform = 6
		case "Hour", "Hours":
			st[i] = ""
			idxRel = i
			closeTime = true
			transform.Type = transformHour
		case "Few", "Couple":
			st[i] = ""
			transform.Transform = 2
		case "Min", "Mins", "Minute", "Minutes":
			st[i] = ""
			idxRel = i
			closeTime = true
			transform.Type = transformMinute
		case "Day", "Days":
			st[i] = ""
			idxRel = i
			transform.Type = transformDay
		case "Week", "Weeks":
			st[i] = ""
			idxRel = i
			transform.Type = transformDay
			transform.Multiplier = 7
		case "Month", "Months":
			st[i] = ""
			idxRel = i
			transform.Type = transformMonth
		case "Year", "Years":
			st[i] = ""
			idxRel = i
			transform.Type = transformYear

		// Remove unnecessary but common expressions like "at", "time",
		// "oclock".
		case "At", "Time", "Oclock", "This", "The":
			st[i] = ""
		case "Noon":
			st[i] = "12PM"
		case "Supper", "Dinner":
			st[i] = "6PM"
		}

		if len(st[i]) > 0 {
			stFull += st[i] + " "
		}
	}
	normalized := strings.TrimRight(stFull, " ")

	var timeEmpty bool
	ts := []time.Time{}
	tme, err := parseTime(normalized)
	if err != nil {
		// Set the hour to 9am
		timeEmpty = true
		tme = time.Now().Round(time.Hour)
		val := 9 - tme.Hour()
		tme = tme.Add(time.Duration(val) * time.Hour)
	}
	if closeTime {
		tme = time.Now().Round(time.Minute)
	}
	ts = append(ts, tme)

	// TODO make more efficient. Handle in switch?
	tloc := timeLocation{loc: loc}
	ctx := &TimeContext{ampm: ampmNoTime, tz: tloc}
	if strings.Contains(normalized, "AM") {
		ctx.ampm = amTime
	}
	if strings.Contains(normalized, "UTC") {
		ctx.tz.utc = true
	}
	for _, ti := range ts {
		ctx = updateContext(ctx, ti, false)
	}

	// Ensure dates are reasonable even in the absence of information.
	// e.g. 2AM should parse to the current year, not 0000
	ctx = completeContext(ctx, t)

	// Loop through a second time to apply the discovered context to each
	// time. Note that this doesn't support context switching,
	// e.g. "5PM CST or PST" or "5PM EST or 6PM PST", which is rare in
	// practice. Future versions may be adapted to support it.
	if ctx.ampm == ampmNoTime {
		halfLen := len(ts)
		// Double size of times for AM/PM
		ts = append(ts, ts...)
		for i := range ts {
			var hour int
			t := ts[i]
			if i < halfLen {
				hour = t.Hour()
			} else {
				hour = t.Hour() + 12
			}
			ts[i] = time.Date(ctx.year,
				ctx.month,
				ctx.day,
				hour,
				t.Minute(),
				t.Second(),
				t.Nanosecond(),
				ctx.tz.loc)
		}
	} else {
		for i := range ts {
			t := ts[i]
			ts[i] = time.Date(ctx.year,
				ctx.month,
				ctx.day,
				t.Hour(),
				t.Minute(),
				t.Second(),
				t.Nanosecond(),
				ctx.tz.loc)
		}
	}

	// If there's no relative transform, we're done.
	if transform.Type == transformInvalid {
		if timeEmpty {
			return []time.Time{}
		}
		if idxRel == 0 {
			return ts
		}
	}

	// Check our idxRel term for the word that preceeds it. If that's a
	// number, e.g. 2 days, then that number is our Transform. Note that
	// this doesn't handle fractional modifiers, like 2.5 days.
	if idxRel > 0 {
		val, err := strconv.Atoi(st[idxRel-1])
		if err == nil {
			transform.Transform = val
		}
	}

	// Apply the transform
	log.Debugf("timeparse: normalized %q. %+v\n", normalized, transform)
	for i := range ts {
		switch transform.Type {
		case transformYear:
			ts[i] = ts[i].AddDate(transform.Transform*transform.Multiplier, 0, 0)
		case transformMonth:
			ts[i] = ts[i].AddDate(0, transform.Multiplier*transform.Transform, 0)
		case transformDay:
			ts[i] = ts[i].AddDate(0, 0, transform.Multiplier*transform.Transform)
		case transformHour:
			ts[i] = ts[i].Add(time.Duration(transform.Transform*transform.Multiplier) * time.Hour)
		case transformMinute:
			ts[i] = ts[i].Add(time.Duration(transform.Transform*transform.Multiplier) * time.Minute)
		}
	}
	log.Debug("timeparse: parsed times", ts)
	return ts
}

func updateContext(orig *TimeContext, t time.Time, day bool) *TimeContext {
	if t.Year() != 0 {
		orig.year = t.Year()
	}
	if day {
		orig.month = t.Month()
		orig.day = t.Day()
	} else {
		if t.Month() != 1 {
			orig.month = t.Month()
		}
		if t.Day() != 1 {
			orig.day = t.Day()
		}
	}
	if t.Location() != nil {
		orig.tz.loc = t.Location()
	}
	if orig.ampm == ampmNoTime && t.Hour() > 12 {
		orig.ampm = pmTime
	}
	return orig
}

func completeContext(ctx *TimeContext, t time.Time) *TimeContext {
	if ctx.year == 0 {
		(*ctx).year = t.Year()
	}
	if ctx.month == 0 {
		(*ctx).month = t.Month()
	}
	if ctx.day == 0 {
		(*ctx).day = t.Day()
	}
	// time.Location.String() defaults to "UTC". The bool tz.utc tracks if
	// that was deliberately UTC.
	if ctx.tz.loc.String() == "UTC" && ctx.tz.utc == false {
		(*ctx).tz.loc = t.Location()
	}
	return ctx
}

func loadLocation(l string) *time.Location {
	loc, err := time.LoadLocation(l)
	if err != nil {
		log.Info("failed to load location.", l)
	}
	return loc
}

// parseTime iterates through all known date formats on a normalized time
// string, using Golang's standard lib to do the heavy lifting.
//
// TODO This is a brute-force, "dumb" method of determining the time format and
// should be improved.
func parseTime(t string) (time.Time, error) {
	for _, tf := range timeFormats {
		time, err := time.Parse(tf, t)
		if err == nil {
			log.Debug("timeparse: found format", tf)
			return time, nil
		}
	}
	return time.Time{}, ErrInvalidTimeFormat
}
