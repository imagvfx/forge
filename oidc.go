package forge

// OIDC stands for open id connect.
type OIDC struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HostDomain   string
}
