package dockercompose

import (
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
)

var _ provider.Provider = &DockerCompose{}

type DockerCompose struct{}

func (d *DockerCompose) Name() string {
	return "docker-compose"
}

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
