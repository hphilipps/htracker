package fetch

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFetch(t *testing.T) {

	urlStr1 := "/some/unknown/site"
	resp1 := "Site not found!"
	urlStr2 := "/site/static"
	resp2 := "Static Content"

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

	c := http.Client{}

	url1, err := url.Parse(server.URL + urlStr1)
	if err != nil {
		t.Fatalf("Could not parse URL %s: %v", urlStr1, err)
	}

	body, err := Fetch(url1, &c)
	if !errors.Is(err, ErrStatus) {
		t.Fatalf("Expected HTTP status error, got err: %v", err)
	}

	if want := resp1; want != string(body) {
		t.Fatalf("Expected body: %s, Got: %s", want, string(body))
	}

	url2, err := url.Parse(server.URL + urlStr2)
	if err != nil {
		t.Fatalf("Could not parse URL %s: %v", urlStr2, err)
	}

	body, err = Fetch(url2, &c)
	if err != nil {
		t.Fatalf("Fetching %s: got err: %v", urlStr2, err)
	}

	if want := resp2; want != string(body) {
		t.Fatalf("Expected body: %s, Got: %s", want, string(body))
	}
}
