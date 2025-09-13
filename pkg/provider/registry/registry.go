// Package registry manages the collection of available providers.
package registry

import (
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider/dockercompose"
)

// Providers contains all available providers registered in the system.
var Providers = map[string]provider.Provider{
	"docker-compose": &dockercompose.DockerCompose{},
}

// GetProvider returns the provider with the specified name.
func GetProvider(name string) provider.Provider {
	return Providers[name]
}
