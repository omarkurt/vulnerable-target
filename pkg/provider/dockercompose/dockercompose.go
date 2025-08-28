package dockercompose

import (
	"context"

	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
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

	composeService := compose.NewComposeService(dockerCli)
	ctx := context.Background()

	err = composeService.Down(ctx, project.Name, api.DownOptions{
		RemoveOrphans: true,
		Volumes:       true,
	})
	if err != nil {
		return err
	}

	return nil
}
