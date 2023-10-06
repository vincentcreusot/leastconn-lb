package server

type AuthScheme struct {
	allowedClients map[string][]string // client ID -> allowed upstreams
}

func NewAuthScheme() *AuthScheme {
	// TODO should go into config, hardcoding for now
	m := map[string][]string{
		"client1.lb.com": {"localhost:9801", "localhost:9802"},
		"client2.lb.com": {"localhost:9802"},
	}
	return &AuthScheme{
		allowedClients: m,
	}
}

func (a *AuthScheme) GetAllowedUpstreams(clientID string) []string {
	allowed, ok := a.allowedClients[clientID]
	if !ok {
		return nil
	}
	return allowed
}
