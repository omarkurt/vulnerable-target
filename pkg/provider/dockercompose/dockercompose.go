package dockercompose

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/happyhackingspace/vulnerable-target/internal/file"
	"github.com/happyhackingspace/vulnerable-target/pkg/options"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
	"github.com/rs/zerolog/log"
)

var _ provider.Provider = &DockerCompose{}

type DockerCompose struct{}

func (d *DockerCompose) Name() string {
	return "docker-compose"
}

func (d *DockerCompose) Start() {
	options := options.GetOptions()
	template := templates.Templates[options.TemplateID]
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

func (d *DockerCompose) Stop() {
	options := options.GetOptions()
	template := templates.Templates[options.TemplateID]
	composeContent := template.Providers["docker_compose"].Content

	composeFilePath, err := file.CreateTempFile(composeContent, "docker-compose.yml")
	if err != nil {
		log.Fatal().Msgf("%v", err)
	}

	downCmd := exec.Command("docker", "compose", "-f", composeFilePath, "-p", fmt.Sprintf("vt-compose-%s", template.ID), "down")
	downCmd.Stdout = os.Stdout
	downCmd.Stderr = os.Stderr

	err = downCmd.Run()
	if err != nil {
		log.Fatal().Msgf("%v", err)
	}
}
