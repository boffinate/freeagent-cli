package cli

type Runtime struct {
	ConfigPath string
	Profile    string
	Sandbox    bool
	BaseURL    string
	Subdomain  string
	JSONOutput bool
}
