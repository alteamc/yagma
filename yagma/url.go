package yagma

import (
	"net/url"
)

const (
	mojangAPI           = "https://api.mojang.com"
	mojangSessionServer = "https://sessionserver.mojang.com"
)

// BaseURL holds values of base URL used in requests made to Mojang API.
type BaseURL struct {
	api           string
	sessionServer string
}

// NewMojangBaseURL constructs a new BaseURL with default Mojang API base URL.
func NewMojangBaseURL() *BaseURL {
	return NewBaseURL(mojangAPI, mojangSessionServer)
}

// NewBaseURL constructs a new BaseURL with custom base URL.
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

// API constructs a new endpoint URL with API base URL prefix, endpoint path and query if it's not empty.
func (u *BaseURL) API(endpoint string, query url.Values) string {
	return getQueryURL(u.api+endpoint, query)
}

// SessionServer constructs a new endpoint URL with SessionServer base URL prefix, endpoint path and
// query if it's not empty.
func (u *BaseURL) SessionServer(endpoint string, query url.Values) string {
	return getQueryURL(u.sessionServer+endpoint, query)
}
