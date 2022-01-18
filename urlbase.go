package yagma

const (
	mojangAPI           = "https://api.mojang.com"
	mojangSessionServer = "https://sessionserver.mojang.com"
)

type URLBase struct {
	mojangAPI     string
	sessionServer string
}

func NewMojangURLBase() *URLBase {
	return NewURLBase(mojangAPI, mojangSessionServer)
}

func NewURLBase(mojangAPI string, sessionServer string) *URLBase {
	return &URLBase{
		mojangAPI:     mojangAPI,
		sessionServer: sessionServer,
	}
}
