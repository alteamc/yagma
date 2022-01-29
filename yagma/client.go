package yagma

import (
	"net/http"
)

// Client is a core of Yagma library.
type Client struct {
	httpClient *http.Client
	baseURL    *BaseURL
}

// New constructs new API Client with default options.
func New() *Client {
	return NewWithOptions(
		WithHTTPClient(&http.Client{}),
		WithURLBase(NewMojangBaseURL()),
	)
}

// NewWithOptions constructs new API Client with custom HTTP client and BaseURL.
func NewWithOptions(opts ...Option) *Client {
	c := &Client{httpClient: &http.Client{}}
	for _, opt := range opts {
		opt.configure(c)
	}
	return c
}

// Option configures Client in a certain manner.
type Option struct{ configure optionFunc }

type optionFunc func(*Client)

// WithHTTPClient returns an Option for use in NewWithOptions method for construction of a Client
// with custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return Option{func(c *Client) {
		c.httpClient = client
	}}
}

// WithURLBase returns an Option for use in NewWithOptions method for construction of a Client
// with custom BaseURL.
func WithURLBase(urlBase *BaseURL) Option {
	return Option{func(c *Client) {
		c.baseURL = urlBase
	}}
}
