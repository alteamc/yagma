package yagma

import (
	"net/url"
)

const (
	mojangAPI           = "https://api.mojang.com"
	mojangSessionServer = "https://sessionserver.mojang.com"
)

type BaseURL struct {
	client        *Client
	api           string
	sessionServer string
}

func NewMojangBaseURL() *BaseURL {
	return NewBaseURL(mojangAPI, mojangSessionServer)
}

func NewBaseURL(api string, sessionServer string) *BaseURL {
	return &BaseURL{
		api:           api,
		sessionServer: sessionServer,
	}
}

func getQueryURL(base string, query url.Values) string {
	if len(query) == 0 {
		return base
	} else {
		return base + "?" + query.Encode()
	}
}

func (u *BaseURL) API(endpoint string, query url.Values) string {
	return getQueryURL(u.api+endpoint, query)
}

func (u *BaseURL) SessionServer(endpoint string, query url.Values) string {
	return getQueryURL(u.sessionServer+endpoint, query)
}
