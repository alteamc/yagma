package yagma

const (
	MojangAPI           = "https://api.mojang.com"
	MojangSessionServer = "https://sessionserver.mojang.com"
)

type URLBase struct {
	mojangAPI     string
	sessionServer string
}

func NewMojangURLBase() *URLBase {
	return NewURLBase("", "")
}

func NewURLBase(mojangAPI string, sessionServer string) *URLBase {
	if mojangAPI == "" {
		mojangAPI = MojangAPI
	}

	if sessionServer == "" {
		sessionServer = MojangSessionServer
	}

	return &URLBase{
		mojangAPI:     mojangAPI,
		sessionServer: sessionServer,
	}
}
