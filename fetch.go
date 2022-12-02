package htracker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var ErrStatus = fmt.Errorf("HTTP status code not ok")

type Fetcher struct {
	url    *url.URL
	client *http.Client
}

type FetcherOption func(*Fetcher)

// NewFetcher is returning a new Fetcher.
func NewFetcher(url *url.URL, client *http.Client, options ...FetcherOption) *Fetcher {

	fetcher := &Fetcher{
		url:    url,
		client: client,
	}

	for _, o := range options {
		o(fetcher)
	}

	return fetcher
}

// Stream is returning the request body as io.ReadCloser.
//
// The caller is responsible to close the returned body if no error was returned.
func (f *Fetcher) Stream() (body io.ReadCloser, err error) {
	resp, err := f.client.Get(f.url.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("Fetch: Status code: %d - %w", resp.StatusCode, ErrStatus)
	}

	return resp.Body, nil
}

// Fetch is returning the whole content of the body of the site associated to the Fetcher.
func (f *Fetcher) Fetch() (content []byte, err error) {
	body, err := f.Stream()
	if err != nil {
		return nil, err
	}
	defer body.Close()

	content, err = io.ReadAll(body)
	if err != nil {
		return content, err
	}

	return content, nil
}
