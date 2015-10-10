package twilio

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestException(t *testing.T) {
	data := `{
		"status": 400,
		"message": "No to number is specified",
		"code": 21201,
		"more_info": "http:\/\/www.twilio.com\/docs\/errors\/21201"
	}`

	ex := new(Exception)
	err := json.Unmarshal([]byte(data), &ex)
	assert.Nil(t, err)

	want := &Exception{
		Status:   400,
		Message:  "No to number is specified",
		Code:     21201,
		MoreInfo: "http://www.twilio.com/docs/errors/21201",
	}

	assert.Equal(t, ex, want)
}

func TestException_Error(t *testing.T) {
	ex := &Exception{
		Status:   400,
		Message:  "No to number is specified",
		Code:     21201,
		MoreInfo: "http://www.twilio.com/docs/errors/21201",
	}

	want := "21201: No to number is specified"
	assert.Equal(t, ex.Error(), want)
}
