package twilio

import (
	"encoding/json"
	"github.com/subosito/figo"
	"io/ioutil"
	"net/http"
	"net/url"
)

func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	exception := new(Exception)
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && data != nil {
		json.Unmarshal(data, &exception)
	}

	return exception
}

func structToUrlValues(i interface{}) url.Values {
	v := url.Values{}
	m := figo.StructToMapString(i)
	for k, s := range m {
		switch {
		case len(s) == 1:
			v.Set(k, s[0])
		case len(s) > 1:
			for i := range s {
				v.Add(k, s[i])
			}
		}
	}

	return v
}
