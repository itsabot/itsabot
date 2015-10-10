package twilio

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPagination(t *testing.T) {
	data := `{
		"start": 0,
		"total": 261,
		"num_pages": 6,
		"page": 0,
		"page_size": 50,
		"end": 49,
		"uri": "/2010-04-01/Accounts/AC5ef87/Messages.json",
		"first_page_uri": "/2010-04-01/Accounts/AC5ef87/Messages.json?Page=0&PageSize=50",
		"last_page_uri": "/2010-04-01/Accounts/AC5ef87/Messages.json?Page=5&PageSize=50",
		"next_page_uri": "/2010-04-01/Accounts/AC5ef87/Messages.json?Page=1&PageSize=50",
		"previous_page_uri": null
	}`

	p := new(Pagination)
	err := json.Unmarshal([]byte(data), &p)
	if err != nil {
		t.Errorf("json.Unmarshal Pagination returned an error %+v", err)
	}

	want := &Pagination{
		Start:           0,
		Total:           261,
		NumPages:        6,
		Page:            0,
		PageSize:        50,
		End:             49,
		Uri:             "/2010-04-01/Accounts/AC5ef87/Messages.json",
		FirstPageUri:    "/2010-04-01/Accounts/AC5ef87/Messages.json?Page=0&PageSize=50",
		LastPageUri:     "/2010-04-01/Accounts/AC5ef87/Messages.json?Page=5&PageSize=50",
		NextPageUri:     "/2010-04-01/Accounts/AC5ef87/Messages.json?Page=1&PageSize=50",
		PreviousPageUri: "",
	}

	if !reflect.DeepEqual(p, want) {
		t.Errorf("Pagination returned %+v, want %+v", p, want)
	}
}
