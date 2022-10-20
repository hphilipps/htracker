package fetch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

var ErrStatus = fmt.Errorf("HTTP status code not ok")

func Fetch(url *url.URL, client *http.Client) (body []byte, err error) {
	resp, err := client.Get(url.String())
	if err != nil {
		return body, err
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return body, fmt.Errorf("Fetch: Status code: %d. %w", resp.StatusCode, ErrStatus)
	}

	return body, nil
}
