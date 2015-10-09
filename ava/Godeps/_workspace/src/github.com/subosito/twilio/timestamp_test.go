package twilio

import (
	"testing"
	"time"
)

func TestTimestamp_IsZero(t *testing.T) {
	m := &Timestamp{}
	if !m.IsZero() {
		t.Error("Timestamp.IsZero() should be true")
	}

	p := &Timestamp{Time: time.Date(2013, 4, 87, 21, 10, 56, 0, time.UTC)}
	if p.IsZero() {
		t.Error("Timestamp.IsZero() should be false")
	}
}

func TestTimestamp_UnmarshalJSON_string(t *testing.T) {
	m := &Timestamp{}
	err := m.UnmarshalJSON([]byte("Wed, 18 Aug 2010 20:01:40 +0000"))
	if err != nil {
		t.Error("Price.UnmarshalJSON returned an error %q", err)
	}

	want := parseTimestamp("Wed, 18 Aug 2010 20:01:40 +0000")
	if !m.Equal(want) {
		t.Errorf("Time.UnmarshalJSON returned %+v, want %+v", m, want)
	}
}

func TestTimestamp_UnmarshalJSON_badString(t *testing.T) {
	m := &Timestamp{}
	err := m.UnmarshalJSON([]byte("foo/02/03"))
	if err != nil {
		t.Error("Price.UnmarshalJSON returned an error %q", err)
	}

	want := Timestamp{}
	if !m.Equal(want) {
		t.Errorf("Time.UnmarshalJSON returned %+v, want %+v", m, want)
	}
}
