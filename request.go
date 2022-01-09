package main

import (
	"context"
	"io"
	"net/http"
)

func (c *Client) sendHTTPReq(ctx context.Context, method, url string, header http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = header

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
