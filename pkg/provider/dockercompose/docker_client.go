package dockercompose

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// DockerClient wraps the Docker API client with helper methods
type DockerClient struct {
	client *client.Client
}

// NewDockerClient creates a new Docker client
func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &DockerClient{client: cli}, nil
}

// GetContainersByProject returns all containers for a given project
func (dc *DockerClient) GetContainersByProject(ctx context.Context, projectName string) ([]types.Container, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := dc.client.ContainerList(ctx, types.ContainerListOptions{
		Filters: filterArgs,
		All:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// GetContainerStatus returns the status of containers for a project
func (dc *DockerClient) GetContainerStatus(ctx context.Context, projectName string) (map[string]ContainerStatus, error) {
	containers, err := dc.GetContainersByProject(ctx, projectName)
	if err != nil {
		return nil, err
	}

	status := make(map[string]ContainerStatus)
	for _, container := range containers {
		serviceName := container.Labels["com.docker.compose.service"]
		if serviceName == "" {
			continue
		}

		// Get detailed container info
		inspect, err := dc.client.ContainerInspect(ctx, container.ID)
		if err != nil {
			log.Warn().Err(err).Str("container", container.ID).Msg("Failed to inspect container")
			continue
		}

		cs := ContainerStatus{
			ID:      container.ID[:12],
			Name:    strings.TrimPrefix(container.Names[0], "/"),
			Service: serviceName,
			State:   container.State,
			Status:  container.Status,
			Created: time.Unix(container.Created, 0),
		}

		// Check health status
		if inspect.State.Health != nil {
			cs.Health = inspect.State.Health.Status
		}

		// Get exposed ports
		for _, port := range container.Ports {
			if port.PublicPort != 0 {
				cs.Ports = append(cs.Ports, fmt.Sprintf("%d:%d/%s", 
					port.PublicPort, port.PrivatePort, port.Type))
			}
		}

		status[serviceName] = cs
	}

	return status, nil
}

// WaitForHealthy waits for all containers in a project to become healthy
func (dc *DockerClient) WaitForHealthy(ctx context.Context, projectName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for containers to become healthy")
			}

			containers, err := dc.GetContainersByProject(ctx, projectName)
			if err != nil {
				return err
			}

			allHealthy := true
			for _, container := range containers {
				if container.State != "running" {
					allHealthy = false
					break
				}

				// Check health if configured
				inspect, err := dc.client.ContainerInspect(ctx, container.ID)
				if err != nil {
					continue
				}

				if inspect.State.Health != nil && inspect.State.Health.Status != "healthy" {
					allHealthy = false
					log.Debug().
						Str("container", container.Names[0]).
						Str("health", inspect.State.Health.Status).
						Msg("Container not yet healthy")
					break
				}
			}

			if allHealthy && len(containers) > 0 {
				return nil
			}
		}
	}
}

// StopContainers stops all containers for a project
func (dc *DockerClient) StopContainers(ctx context.Context, projectName string, timeout time.Duration) error {
	containers, err := dc.GetContainersByProject(ctx, projectName)
	if err != nil {
		return err
	}

	// Convert timeout to seconds, Docker API expects seconds as int
	timeoutSeconds := int(timeout.Seconds())
	
	for _, c := range containers {
		if c.State == "running" {
			log.Debug().Str("container", c.Names[0]).Msg("Stopping container")
			// ContainerStop expects container.StopOptions
			options := container.StopOptions{}
			if timeoutSeconds > 0 {
				options.Timeout = &timeoutSeconds
			}
			if err := dc.client.ContainerStop(ctx, c.ID, options); err != nil {
				log.Warn().Err(err).Str("container", c.Names[0]).Msg("Failed to stop container")
			}
		}
	}

	return nil
}

// RemoveContainers removes all containers for a project
func (dc *DockerClient) RemoveContainers(ctx context.Context, projectName string, removeVolumes bool) error {
	containers, err := dc.GetContainersByProject(ctx, projectName)
	if err != nil {
		return err
	}

	for _, container := range containers {
		log.Debug().Str("container", container.Names[0]).Msg("Removing container")
		if err := dc.client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
			RemoveVolumes: removeVolumes,
			Force:         true,
		}); err != nil {
			log.Warn().Err(err).Str("container", container.Names[0]).Msg("Failed to remove container")
		}
	}

	return nil
}

// GetNetworksByProject returns all networks for a project
func (dc *DockerClient) GetNetworksByProject(ctx context.Context, projectName string) ([]types.NetworkResource, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	networks, err := dc.client.NetworkList(ctx, types.NetworkListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	return networks, nil
}

// RemoveNetworks removes all networks for a project
func (dc *DockerClient) RemoveNetworks(ctx context.Context, projectName string) error {
	networks, err := dc.GetNetworksByProject(ctx, projectName)
	if err != nil {
		return err
	}

	for _, network := range networks {
		// Skip default networks
		if network.Name == "bridge" || network.Name == "host" || network.Name == "none" {
			continue
		}

		log.Debug().Str("network", network.Name).Msg("Removing network")
		if err := dc.client.NetworkRemove(ctx, network.ID); err != nil {
			log.Warn().Err(err).Str("network", network.Name).Msg("Failed to remove network")
		}
	}

	return nil
}

// StreamContainerLogs streams logs from a container
func (dc *DockerClient) StreamContainerLogs(ctx context.Context, containerID string, writer io.Writer) error {
	options := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	logs, err := dc.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	_, err = io.Copy(writer, logs)
	return err
}

// CreateNetwork creates a network for the project
func (dc *DockerClient) CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error) {
	networkCreate := types.NetworkCreate{
		Driver:     "bridge",
		Attachable: true,
		Labels:     labels,
	}

	response, err := dc.client.NetworkCreate(ctx, name, networkCreate)
	if err != nil {
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	return response.ID, nil
}

// CreateContainer creates a container with the given configuration
func (dc *DockerClient) CreateContainer(ctx context.Context, config *container.Config, 
	hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig, name string) (string, error) {
	
	resp, err := dc.client.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a container
func (dc *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	if err := dc.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	return nil
}

// PullImage pulls a Docker image
func (dc *DockerClient) PullImage(ctx context.Context, image string) error {
	reader, err := dc.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	defer reader.Close()

	// Read the output to ensure the pull completes
	_, err = io.Copy(io.Discard, reader)
	return err
}

// ImageExists checks if an image exists locally
func (dc *DockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	_, _, err := dc.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Close closes the Docker client connection
func (dc *DockerClient) Close() error {
	return dc.client.Close()
}

// ContainerStatus represents the status of a container
type ContainerStatus struct {
	ID      string
	Name    string
	Service string
	State   string
	Status  string
	Health  string
	Created time.Time
	Ports   []string
}
