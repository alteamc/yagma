package yagma

import (
	"net/http"
)

// Client

type Client struct {
	httpClient *http.Client
	urlBase    *URLBase
}

func New() *Client {
	return NewWithOptions(
		WithHTTPClient(&http.Client{}),
		WithURLBase(NewMojangURLBase()),
	)
}

func NewWithOptions(opts ...Option) *Client {
	c := &Client{httpClient: &http.Client{}}
	for _, opt := range opts {
		opt.configure(c)
	}
	return c
}

// Options

type optionFunc func(*Client)
type Option struct{ configure optionFunc }

func WithHTTPClient(client *http.Client) Option {
	return Option{func(c *Client) {
		c.httpClient = client
	}}
}

func WithURLBase(urlBase *URLBase) Option {
	return Option{func(c *Client) {
		c.urlBase = urlBase
	}}
}
