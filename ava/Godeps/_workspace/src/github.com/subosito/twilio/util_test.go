package twilio

import (
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestCheckResponse(t *testing.T) {
	res := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader(`{"status": 400, "code": 21201, "message": "invalid parameter"}`)),
	}

	err := CheckResponse(res).(*Exception)

	if err == nil {
		t.Error("CheckResponse expected error response")
	}

	want := &Exception{
		Status:  400,
		Code:    21201,
		Message: "invalid parameter",
	}

	if !reflect.DeepEqual(err, want) {
		t.Errorf("Exception = %#v, want %#v", err, want)
	}
}
