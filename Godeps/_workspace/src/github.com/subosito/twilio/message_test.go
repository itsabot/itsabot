package twilio

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestMessage_IsSent(t *testing.T) {
	m := &Message{Status: "sent"}
	if !m.IsSent() {
		t.Error("Message.IsSent() should be true")
	}

	s := &Message{Status: "queued"}
	if s.IsSent() {
		t.Error("Message.IsSent() should be false")
	}
}

func TestMessageParams_Validates(t *testing.T) {
	m := &MessageParams{}

	err := m.Validates()
	if err == nil {
		t.Error("Message.Validates expected an error to be returned")
	}
}

func TestMessageService_Create(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages")

	output := `{
		"sid": "abcdef",
		"num_media": "1",
		"price": "0.74",
		"date_sent": null
	}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, output)
	})

	v := url.Values{
		"From":     {"+14158141829"},
		"To":       {"+15558675309"},
		"Body":     {"I love you <3"},
		"MediaUrl": {"http://www.example.com/hearts.png"},
	}

	m, _, err := client.Messages.Create(v)

	if err != nil {
		t.Errorf("Message.Send returned error: %q", err)
	}

	want := &Message{
		Sid:      "abcdef",
		NumMedia: 1,
		Price:    0.74,
		DateSent: Timestamp{},
	}

	if !reflect.DeepEqual(m, want) {
		t.Errorf("Message.Create() returned %+v, want %+v", m, want)
	}
}

func TestMessageService_Send(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages")

	output := `{
		"sid": "abcdef",
		"num_media": "1",
		"price": "0.74",
		"date_sent": null
	}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, output)
	})

	params := MessageParams{
		Body:     "I love you <3",
		MediaUrl: []string{"http://www.example.com/hearts.png"},
	}

	m, _, err := client.Messages.Send("+14158141829", "+15558675309", params)

	if err != nil {
		t.Errorf("Message.Send returned error: %q", err)
	}

	want := &Message{
		Sid:      "abcdef",
		NumMedia: 1,
		Price:    0.74,
		DateSent: Timestamp{},
	}

	if !reflect.DeepEqual(m, want) {
		t.Errorf("Message.SendSMS returned %+v, want %+v", m, want)
	}
}

func TestMessageService_SendSMS(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages")

	output := `{
		"sid": "abcdef",
		"num_media": "0",
		"price": "0.74",
		"date_created": "Wed, 18 Aug 2010 20:01:40 +0000"
	}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, output)
	})

	m, _, err := client.Messages.SendSMS("+1234567", "+7654321", "Hello!")

	if err != nil {
		t.Errorf("Message.SendSMS returned error: %v", err)
	}

	tm := parseTimestamp("Wed, 18 Aug 2010 20:01:40 +0000")
	want := &Message{
		Sid:         "abcdef",
		NumMedia:    0,
		Price:       0.74,
		DateCreated: tm,
	}

	if !reflect.DeepEqual(m, want) {
		t.Errorf("Message.SendSMS returned %+v, want %+v", m, want)
	}
}

func TestMessageService_Send_requiredParams(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.Messages.Send("+14158141829", "+15558675309", MessageParams{})

	if err == nil {
		t.Error("Send with MessageParams{} expected to be returned an error")
	}
}

func TestMessageService_Send_incompleteParams(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages")

	output := `{
		"status": 400,
		"message": "A 'From' phone number is required.",
		"code": 21603
	}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, output)
	})

	_, r, err := client.Messages.Send("", "+15558675309", MessageParams{Body: "Hello"})

	if r.StatusCode != http.StatusBadRequest {
		t.Errorf("Send() status code = %d, want %d", r.StatusCode, http.StatusBadRequest)
	}

	ex, _ := err.(*Exception)
	want := &Exception{
		Status:  400,
		Message: "A 'From' phone number is required.",
		Code:    21603,
	}

	if !reflect.DeepEqual(ex, want) {
		t.Errorf("Message.SendSMS returned %+v, want %+v", ex, want)
	}
}

func TestMessageService_Get(t *testing.T) {
	setup()
	defer teardown()

	sid := "MM90c6fc909d8504d45ecdb3a3d5b3556e"
	u := client.EndPoint("Messages", sid)

	output := `{
		"account_sid": "AC5ef8732a3c49700934481addd5ce1659",
		"api_version": "2010-04-01",
		"body": "I love you <3",
		"num_segments": "1",
		"num_media": "1",
		"date_created": "Wed, 18 Aug 2010 20:01:40 +0000",
		"date_sent": null,
		"date_updated": "Wed, 18 Aug 2010 20:01:40 +0000",
		"direction": "outbound-api",
		"from": "+14158141829",
		"price": null,
		"sid": "MM90c6fc909d8504d45ecdb3a3d5b3556e",
		"status": "queued",
		"to": "+15558675309",
		"uri": "/2010-04-01/Accounts/AC5ef87/Messages/MM90c6.json"
	}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, output)
	})

	m, _, err := client.Messages.Get(sid)

	if err != nil {
		t.Errorf("Message.SendSMS returned error: %v", err)
	}

	tm := parseTimestamp("Wed, 18 Aug 2010 20:01:40 +0000")
	want := &Message{
		AccountSid:  "AC5ef8732a3c49700934481addd5ce1659",
		ApiVersion:  "2010-04-01",
		Body:        "I love you <3",
		NumSegments: 1,
		NumMedia:    1,
		DateCreated: tm,
		DateSent:    Timestamp{},
		DateUpdated: tm,
		Direction:   "outbound-api",
		From:        "+14158141829",
		Price:       0,
		Sid:         "MM90c6fc909d8504d45ecdb3a3d5b3556e",
		Status:      "queued",
		To:          "+15558675309",
		Uri:         "/2010-04-01/Accounts/AC5ef87/Messages/MM90c6.json",
	}

	if !reflect.DeepEqual(m, want) {
		t.Errorf("Message.SendSMS returned %+v, want %+v", m, want)
	}
}

func TestMessageService_Get_httpError(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages", "abc")

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})

	_, _, err := client.Messages.Get("abc")

	if err == nil {
		t.Error("Expected HTTP 400 errror.")
	}
}

func TestMessageService_List(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages")

	output := `{
		"page": 1,
		"page_size": 50,
		"uri": "foo.json",
		"messages": [{ "sid": "MM90c6fc909d8504d45ecdb3a3d5b3556e" }]
	}`

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, output)
	})

	ml, r, err := client.Messages.List(MessageListParams{})

	if err != nil {
		t.Error("Get() err expected to be nil")
	}

	rwant := Pagination{
		Page:     1,
		PageSize: 50,
		Uri:      "foo.json",
	}

	if !reflect.DeepEqual(r.Pagination, rwant) {
		t.Errorf("response.Pagination returned %+v, want %+v", r.Pagination, rwant)
	}

	want := []Message{
		Message{Sid: "MM90c6fc909d8504d45ecdb3a3d5b3556e"},
	}

	if !reflect.DeepEqual(ml, want) {
		t.Errorf("Message.List returned %+v, want %+v", ml, want)
	}
}

func TestMessageService_List_httpError(t *testing.T) {
	setup()
	defer teardown()

	u := client.EndPoint("Messages")

	mux.HandleFunc(u.String(), func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})

	_, _, err := client.Messages.List(MessageListParams{})

	if err == nil {
		t.Error("Expected HTTP 400 errror.")
	}
}
