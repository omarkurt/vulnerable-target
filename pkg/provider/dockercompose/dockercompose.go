package dockercompose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
)

var _ provider.Provider = &DockerCompose{}

type DockerCompose struct{}

func (d *DockerCompose) Name() string {
	return "docker-compose"
}

func (d *DockerCompose) Start(template *templates.Template) error {
	path := template.Providers["docker-compose"].Path
	var upCmd *exec.Cmd
	if filepath.IsAbs(path) {
		upCmd = exec.Command("docker", "compose", "-f", path, "-p", fmt.Sprintf("vt-compose-%s", template.ID), "up", "-d") // #nosec G204
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		upCmd = exec.Command("docker", "compose", "-f", filepath.Join(wd, "templates", template.ID, path), "-p", fmt.Sprintf("vt-compose-%s", template.ID), "up", "-d") // #nosec G204
	}

	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr

	err := upCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (d *DockerCompose) Stop(template *templates.Template) error {
	path := template.Providers["docker-compose"].Path
	var upCmd *exec.Cmd
	if filepath.IsAbs(path) {
		upCmd = exec.Command("docker", "compose", "-f", path, "-p", fmt.Sprintf("vt-compose-%s", template.ID), "down", "--volumes") // #nosec G204
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		upCmd = exec.Command("docker", "compose", "-f", filepath.Join(wd, "templates", template.ID, path), "-p", fmt.Sprintf("vt-compose-%s", template.ID), "down", "--volumes") // #nosec G204
	}

	upCmd.Stdout = os.Stdout
	upCmd.Stderr = os.Stderr

	err := upCmd.Run()
	if err != nil {
		return err
	}

	return nil
}
