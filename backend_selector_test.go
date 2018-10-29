package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestBackendSelector(t *testing.T) {
	bs := BackendSelector{
		Mapping: map[string]string{
			"mjolnir-slots": "abc",
		},
	}
	for i, tc := range []struct {
		host         string
		body         string
		expectedHost string
		expectedErr  error
	}{
		{
			host:         "example.com",
			body:         sampleSessionBody,
			expectedHost: "abc",
			expectedErr:  nil,
		},
		{
			host:         "example.com",
			body:         "{}",
			expectedHost: "example.com",
			expectedErr:  Err404,
		},
		{
			host:         "example.com",
			body:         `{"game":"unknown"}`,
			expectedHost: "example.com",
			expectedErr:  Err404,
		},
	} {
		req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/path", tc.host), ioutil.NopCloser(bytes.NewReader([]byte(tc.body))))
		if err != nil {
			t.Errorf("%d: %s", i, err.Error())
		}
		if err := bs.ModifyRequest(req); err != tc.expectedErr {
			t.Errorf("%d: unexpected error: %s", i, err.Error())
		}
		if tc.expectedHost != req.URL.Host {
			t.Errorf("%d: unexpected host: %s", i, req.URL.Host)
		}
	}

}

var sampleSessionBody = `{
  "casino_id": "s4",
  "game": "mjolnir-slots",
  "currency": "EUR",
  "user":
  { 
    "id": "3422",
    "email": "gamma.solutions@example.com",
    "firstname": "John",
    "lastname": "Doe",
    "nickname": "John Doe",
    "city": "Bucharest",
    "country": "RO",
    "date_of_birth": "1983-12-19",
    "registered_at": "2017-05-15",
    "gender": "m"
  },
  "locale": "en",
  "ip": "195.222.65.88",
  "balance": 113875,
  "urls":
  { 
    "deposit_url": "http://s4.casino.softswiss.com/accounts/EUR/deposit",
    "return_url": "http://s4.casino.softswiss.com/exit_iframe"
  }
}`
