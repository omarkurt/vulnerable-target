package providers

import (
	"github.com/happyhackingspace/vulnerable-target/internal/config"
	"github.com/happyhackingspace/vulnerable-target/pkg/providers/docker"
	"github.com/happyhackingspace/vulnerable-target/pkg/providers/dockercompose"
	"github.com/rs/zerolog/log"
)

func Start() {
	settings := config.GetSettings()
	switch settings.ProviderName {
	case "docker":
		docker.Run()
	case "docker-compose":
		dockercompose.Run()
	}
	log.Info().Msgf("%s template is running on %s", settings.TemplateID, settings.ProviderName)
}
