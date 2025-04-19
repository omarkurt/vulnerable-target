package registry

import (
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider/dockercompose"
)

var Providers = map[string]provider.Provider{
	"docker-compose": &dockercompose.DockerCompose{},
}
