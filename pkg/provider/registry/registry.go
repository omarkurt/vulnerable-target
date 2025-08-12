package registry

import (
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider/dockercompose"
)

// Providers holds all available providers
var Providers map[string]provider.Provider

// init initializes the provider registry
func init() {
	Providers = map[string]provider.Provider{
		"docker-compose": dockercompose.NewDockerCompose(),
	}
}

// GetProvider returns a provider by name
func GetProvider(name string) provider.Provider {
	return Providers[name]
}

// RegisterProvider registers a new provider
func RegisterProvider(name string, p provider.Provider) {
	if Providers == nil {
		Providers = make(map[string]provider.Provider)
	}
	Providers[name] = p
}
