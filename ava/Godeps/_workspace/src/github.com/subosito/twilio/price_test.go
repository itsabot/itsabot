package twilio

import (
	"reflect"
	"testing"
)

func testPrice(t *testing.T) {
	var p Price
	err := p.UnmarshalJSON([]byte(`0.74`))
	if err != nil {
		t.Error("Price.UnmarshalJSON returned an error %q", err)
	}

	want := 0.74
	if !reflect.DeepEqual(p, want) {
		t.Errorf("Price.UnmarshalJSON returned %+v, want %+v", p, want)
	}
}

func TestPrice_UnmarshalJSON_string(t *testing.T) {
	var p Price
	err := p.UnmarshalJSON([]byte(`"0.74"`))
	if err != nil {
		t.Error("Price.UnmarshalJSON returned an error %q", err)
	}

	want := 0.74
	if float64(p) != want {
		t.Errorf("Price.UnmarshalJSON returned %+v, want %+v", p, want)
	}
}

func TestPrice_UnmarshalJSON_null(t *testing.T) {
	var p Price
	err := p.UnmarshalJSON([]byte(`null`))
	if err != nil {
		t.Error("Price.UnmarshalJSON returned an error %q", err)
	}

	want := 0.0
	if float64(p) != want {
		t.Errorf("Price.UnmarshalJSON returned %+v, want %+v", p, want)
	}
}
