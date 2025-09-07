// Package dockercompose provides Docker Compose provider implementation for managing vulnerable target environments.
package dockercompose

import (
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
)

var _ provider.Provider = &DockerCompose{}

// DockerCompose implements the Provider interface using Docker Compose.
type DockerCompose struct{}

// Name returns the provider name.
func (d *DockerCompose) Name() string {
	return "docker-compose"
}

// Start launches the vulnerable target environment using Docker Compose.
func (d *DockerCompose) Start(template *templates.Template) error {
	dockerCli, err := createDockerCLI()
	if err != nil {
		return err
	}

	project, err := loadComposeProject(*template)
	if err != nil {
		return err
	}

	err = runComposeUp(dockerCli, project)
	if err != nil {
		return err
	}

	return nil
}

// Stop shuts down the vulnerable target environment using Docker Compose.
func (d *DockerCompose) Stop(template *templates.Template) error {
	dockerCli, err := createDockerCLI()
	if err != nil {
		return err
	}

	project, err := loadComposeProject(*template)
	if err != nil {
		return err
	}

	err = runComposeDown(dockerCli, project)
	if err != nil {
		return err
	}

	return nil
}
