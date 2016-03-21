// Package timeparse parses times in strings in a wide variety of formats to
// corresponding time.Times.
package timeparse

import (
	"errors"
	"regexp"
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

var timeFormatsNoDay = []string{
	"15",
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
}

var timeFormatsWithDay = []string{
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

var months = map[string]time.Month{
	"Jan": time.Month(1),
	"Feb": time.Month(2),
	"Mar": time.Month(3),
	"Apr": time.Month(4),
	"May": time.Month(5),
	"Jun": time.Month(6),
	"Jul": time.Month(7),
	"Aug": time.Month(8),
	"Sep": time.Month(9),
	"Oct": time.Month(10),
	"Nov": time.Month(11),
	"Dec": time.Month(12),
}

var days = map[string]time.Weekday{
	"Sun": time.Weekday(0),
	"Mon": time.Weekday(1),
	"Tue": time.Weekday(2),
	"Wed": time.Weekday(3),
	"Thu": time.Weekday(4),
	"Fri": time.Weekday(5),
	"Sat": time.Weekday(6),
}

func normalizeTime(t string) (normalizedTime string, location *time.Location,
	relativeTime bool) {

	r := strings.NewReplacer(
		".", "",
		",", "",
		"(", "",
		")", "",
		"'", "",
		"am", "AM",
		"pm", "PM",
		" Am", "AM",
		" Pm", "PM",
		" AM", "AM",
		" A.M", "AM",
		" P.M", "PM")
	t = r.Replace(t)
	t = strings.Title(t)
	st := strings.Fields(t)
	stFull := ""
	relative := false
	var loc *time.Location
	for i := range st {
		// Normalize days
		switch st[i] {
		case "Monday":
			st[i] = "Mon"
			relative = true
		case "Tuesday", "Tues":
			st[i] = "Tue"
			relative = true
		case "Wednesday":
			st[i] = "Wed"
			relative = true
		case "Thursday", "Thur", "Thurs":
			st[i] = "Thu"
			relative = true
		case "Friday":
			st[i] = "Fri"
			relative = true
		case "Saturday":
			st[i] = "Sat"
			relative = true
		case "Sunday":
			st[i] = "Sun"
			relative = true

		// Normalize months
		case "January":
			st[i] = "Jan"
		case "February":
			st[i] = "Feb"
		case "March":
			st[i] = "Mar"
		case "April":
			st[i] = "Apr"
		case "May":
			// No translation needed for May
		case "June":
			st[i] = "Jun"
		case "July":
			st[i] = "Jul"
		case "August":
			st[i] = "Aug"
		case "September", "Sept":
			st[i] = "Sep"
		case "October":
			st[i] = "Oct"
		case "November":
			st[i] = "Nov"
		case "December":
			st[i] = "Dec"

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

		// Handle relative times
		case "Ago", "Previous", "Prev", "Last", "Next", "Upcoming",
			"This", "Tomorrow", "Yesterday":
			relative = true

		// Remove unnecessary but common expressions
		case "At", "Time", "Oclock":
			st[i] = ""
		}

		if st[i] != "" {
			stFull += st[i] + " "
		}
	}
	normalized := strings.TrimRight(stFull, " ")
	return normalized, loc, relative
}

// Parse a natural language string to determine most likely times based on the
// current system time.
func Parse(nlTimes ...string) ([]time.Time, error) {
	return ParseFromTime(time.Now(), nlTimes...)
}

// ParseFromTime parses a natural language string to determine most likely times
// based on a set time "context." The time context changes the meaning of words
// like "this Tuesday," "next Tuesday," etc.
func ParseFromTime(t time.Time, nlTimes ...string) ([]time.Time, error) {
	var times []time.Time
	tloc := timeLocation{loc: t.Location()}
	ctx := &TimeContext{ampm: ampmNoTime, tz: tloc}
	var ti time.Time
	var rel bool
	for _, nlTime := range nlTimes {
		log.Debug("original:", nlTime)
		var loc *time.Location
		nlTime, loc, rel = normalizeTime(nlTime)
		log.Debug("normalized:", nlTime)
		var day bool
		var err error
		if rel {
			ti = parseTimeRelative(nlTime, t)
		} else if loc == nil {
			ti, day, err = parseTime(nlTime)
		} else {
			ti, day, err = parseTimeInLocation(nlTime, loc)
		}
		if err != nil {
			log.Debug("could not parse time", nlTime, err)
			continue
		}
		if !rel {
			ctx = updateContext(ctx, ti, day)
			if strings.Contains(nlTime, "AM") {
				ctx.ampm = amTime
			}
			if strings.Contains(nlTime, "UTC") {
				ctx.tz.utc = true
			}
		}
		times = append(times, ti)
	}

	// Ensure dates are reasonable even in the absence of information.
	// e.g. 2AM should parse to the current year, not 0000
	if !rel {
		ctx = completeContext(ctx, t)
	}

	// Loop through a second time to apply the discovered context to each
	// time. Note that this doesn't support context switching,
	// e.g. "5PM CST or PST" or "5PM EST or 6PM PST", which is rare in
	// practice. Future versions may be adapted to support it.
	if ctx.ampm == ampmNoTime {
		halfLen := len(times)
		// Double size of times for AM/PM
		times = append(times, times...)
		log.Debug(times)
		for i := range times {
			var hour int
			t := times[i]
			if i < halfLen {
				hour = t.Hour()
			} else {
				hour = t.Hour() + 12
			}
			times[i] = time.Date(ctx.year,
				ctx.month,
				ctx.day,
				hour,
				t.Minute(),
				t.Second(),
				t.Nanosecond(),
				ctx.tz.loc)
		}
	} else {
		for i := range times {
			t := times[i]
			times[i] = time.Date(ctx.year,
				ctx.month,
				ctx.day,
				t.Hour(),
				t.Minute(),
				t.Second(),
				t.Nanosecond(),
				ctx.tz.loc)
		}
	}
	return times, nil
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
		ctx.year = t.Year()
	}
	if ctx.month == 0 {
		ctx.month = t.Month()
	}
	if ctx.day == 0 {
		ctx.day = t.Day()
	}
	// time.Location.String() defaults to "UTC". The bool tz.utc tracks if
	// that was deliberately UTC.
	if ctx.tz.loc.String() == "UTC" && ctx.tz.utc == false {
		ctx.tz.loc = t.Location()
	}
	return ctx
}

func loadLocation(l string) *time.Location {
	loc, err := time.LoadLocation(l)
	if err != nil {
		log.Debug("could not load location", l)
	}
	return loc
}

// parseTime iterates through all known date formats on a normalized time
// string, using Golang's standard lib to do the heavy lifting.
//
// TODO This is a brute-force, "dumb" method of determining the time format and
// should be improved.
func parseTime(t string) (time.Time, bool, error) {
	for _, tf := range timeFormatsNoDay {
		time, err := time.Parse(tf, t)
		if err == nil {
			log.Debug("format", tf)
			return time, noDay, nil
		}
	}
	for _, tf := range timeFormatsWithDay {
		time, err := time.Parse(tf, t)
		if err == nil {
			return time, hasDay, nil
		}
	}
	return time.Time{}, noDay, ErrInvalidTimeFormat
}

func parseTimeInLocation(t string, loc *time.Location) (time.Time, bool, error) {
	for _, tf := range timeFormatsNoDay {
		time, err := time.ParseInLocation(tf, t, loc)
		if err == nil {
			return time, noDay, nil
		}
	}
	for _, tf := range timeFormatsWithDay {
		time, err := time.ParseInLocation(tf, t, loc)
		if err == nil {
			return time, hasDay, nil
		}
	}
	return time.Time{}, noDay, ErrInvalidTimeFormat
}

func parseTimeRelative(nlTime string, t time.Time) time.Time {
	hour := t.Hour()
	day := t.Day()
	month := t.Month()
	year := t.Year()
	futureTime := 1
	if strings.Contains(nlTime, "Tomorrow") {
		day++
	}
	if strings.Contains(nlTime, "Yesterday") {
		day--
	}
	if strings.Contains(nlTime, "Month") {
		// TODO handle relative months
		// "2 months", "2 months ago" format
		// "next month", "last month" format
	}
	r := regexp.MustCompile(`(Ago|Prev|Previous|Last)\s`)
	if r.FindString(nlTime) != "" {
		futureTime = -1
	}
	r = regexp.MustCompile(`(Sun|Mon|Tue|Wed|Thu|Fri|Sat)\s`)
	weekday := r.FindString(nlTime)
	if weekday != "" {
		if days[weekday] != t.Weekday() {
			day = day - int(days[weekday]+t.Weekday()) + 7
			if futureTime == -1 {
				day -= 7
			}
		} else {
			day += 7
		}
	}
	r = regexp.MustCompile(
		`(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s`)
	m := r.FindString(nlTime)
	if m != "" {
		if months[m] > month {
			month = months[m]
			if futureTime == -1 {
				month -= 12
			}
		} else if months[m] < month {
			month = months[m] + 12
			if futureTime == -1 {
				month -= 12
			}
		} else {
			month += 12
		}
	}
	return time.Date(
		year,
		month,
		day,
		hour,
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		t.Location())
}
