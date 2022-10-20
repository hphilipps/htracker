package htracker

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFetcher_Fetch(t *testing.T) {

	urlStr1 := "/some/unknown/site"
	resp1 := "Site not found!"
	urlStr2 := "/site/static"
	resp2 := "Static Content"

	tests := []struct {
		name     string
		urlStr   string
		err      error
		response string
	}{
		{name: "not found", urlStr: urlStr1, err: ErrStatus, response: resp1},
		{name: "static site", urlStr: urlStr2, err: nil, response: resp2},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case urlStr1:

			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(resp1))
			return

		case urlStr2:
			w.Write([]byte(resp2))

		default:
			w.Write([]byte(fmt.Sprintf("Some other content. Request path was %s", r.URL.Path)))
		}
	}))

	defer server.Close()

	c := &http.Client{}

	for _, tc := range tests {
		url, err := url.Parse(server.URL + tc.urlStr)
		if err != nil {
			t.Fatalf("Could not parse URL %s: %v", tc.urlStr, err)
		}

		f := NewFetcher(url, c)

		body, err := f.Fetch()

		if want, got := tc.err, err; !errors.Is(got, want) {
			t.Fatalf("%s: Expected error %v, got err: %v", tc.name, want, got)
		}

		if err == nil {
			if want, got := tc.response, string(body); want != got {
				t.Fatalf("%s: Expected content %s, got %s", tc.name, want, got)
			}
		}
	}
}
