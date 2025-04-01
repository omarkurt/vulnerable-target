package dockercompose

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/happyhackingspace/vulnerable-target/internal/config"
	"github.com/happyhackingspace/vulnerable-target/internal/file"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
	"github.com/rs/zerolog/log"
)

func Run() {
	settings := config.GetSettings()
	template := templates.Templates[settings.TemplateID]
	composeContent := template.Providers["docker_compose"].Content

	composeFilePath, err := file.CreateTempFile(composeContent, "docker-compose.yml")
	if err != nil {
		log.Fatal().Msgf("%v", err)
	}

	upCmd := exec.Command("docker", "compose", "-f", composeFilePath, "-p", fmt.Sprintf("vt-compose-%s", template.ID), "up", "-d")
	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr

	err = upCmd.Run()
	if err != nil {
		log.Fatal().Msgf("%v", err)
	}
}
