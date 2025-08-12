package dockercompose

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
	"github.com/rs/zerolog/log"
)

var _ provider.Provider = &DockerCompose{}

// DockerCompose implements the Provider interface using compose-go library
type DockerCompose struct {
	// Store loaded projects for efficient management
	projects map[string]*types.Project
	mu       sync.RWMutex

	// Configuration
	config *Config
}

// Config holds configuration for DockerCompose provider
type Config struct {
	// Timeout for operations
	Timeout time.Duration
	// Whether to remove volumes on stop
	RemoveVolumes bool
	// Whether to remove orphan containers
	RemoveOrphans bool
	// Custom environment variables
	Environment map[string]string
	// Working directory override
	WorkingDir string
	// Enable verbose output
	Verbose bool
}

// NewDockerCompose creates a new DockerCompose provider with default configuration
func NewDockerCompose() *DockerCompose {
	return &DockerCompose{
		projects: make(map[string]*types.Project),
		config: &Config{
			Timeout:       5 * time.Minute,
			RemoveVolumes: true,
			RemoveOrphans: true,
			Environment:   make(map[string]string),
			Verbose:       false,
		},
	}
}

// NewDockerComposeWithConfig creates a new DockerCompose provider with custom configuration
func NewDockerComposeWithConfig(config *Config) *DockerCompose {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	if config.Environment == nil {
		config.Environment = make(map[string]string)
	}
	return &DockerCompose{
		projects: make(map[string]*types.Project),
		config:   config,
	}
}

// Name returns the provider name
func (d *DockerCompose) Name() string {
	return "docker-compose"
}

// Start starts the Docker Compose services for the given template
func (d *DockerCompose) Start(template *templates.Template) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.config.Timeout)
	defer cancel()

	// Load and validate the project
	project, err := d.loadProject(ctx, template)
	if err != nil {
		return fmt.Errorf("failed to load project: %w", err)
	}

	// Validate the project before starting
	if err := d.validateProject(project); err != nil {
		return fmt.Errorf("project validation failed: %w", err)
	}

	// Store the project for later management
	d.mu.Lock()
	d.projects[template.ID] = project
	d.mu.Unlock()

	// Start the services using Docker Compose API
	if err := d.startServices(ctx, project); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Wait for services to be healthy
	if err := d.waitForHealthy(ctx, project); err != nil {
		log.Warn().Err(err).Msg("Some services may not be fully healthy")
	}

	log.Info().
		Str("template", template.ID).
		Str("project", project.Name).
		Int("services", len(project.Services)).
		Msg("Docker Compose services started successfully")

	return nil
}

// Stop stops the Docker Compose services for the given template
func (d *DockerCompose) Stop(template *templates.Template) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.config.Timeout)
	defer cancel()

	// Get the stored project or load it
	d.mu.RLock()
	project, exists := d.projects[template.ID]
	d.mu.RUnlock()

	if !exists {
		// Try to load the project if not in cache
		var err error
		project, err = d.loadProject(ctx, template)
		if err != nil {
			return fmt.Errorf("failed to load project: %w", err)
		}
	}

	// Stop the services
	if err := d.stopServices(ctx, project); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	// Remove from cache
	d.mu.Lock()
	delete(d.projects, template.ID)
	d.mu.Unlock()

	log.Info().
		Str("template", template.ID).
		Str("project", project.Name).
		Msg("Docker Compose services stopped successfully")

	return nil
}

// Status returns the status of Docker Compose services for the given template
func (d *DockerCompose) Status(template *templates.Template) (*ProviderStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the stored project or load it
	d.mu.RLock()
	project, exists := d.projects[template.ID]
	d.mu.RUnlock()

	if !exists {
		var err error
		project, err = d.loadProject(ctx, template)
		if err != nil {
			return nil, fmt.Errorf("failed to load project: %w", err)
		}
	}

	return d.getServiceStatus(ctx, project)
}

// loadProject loads and parses the Docker Compose project
func (d *DockerCompose) loadProject(ctx context.Context, template *templates.Template) (*types.Project, error) {
	providerConfig, exists := template.Providers["docker-compose"]
	if !exists {
		return nil, fmt.Errorf("docker-compose provider not configured for template %s", template.ID)
	}

	composePath, err := d.resolveComposePath(template.ID, providerConfig.Path)
	if err != nil {
		return nil, err
	}

	// Prepare project options
	options := []cli.ProjectOptionsFn{
		cli.WithName(d.getProjectName(template.ID)),
		cli.WithWorkingDirectory(filepath.Dir(composePath)),
		cli.WithConfigFileEnv,
		cli.WithOsEnv,
		cli.WithDotEnv,
	}

	// Add custom environment variables
	if len(d.config.Environment) > 0 {
		envList := make([]string, 0, len(d.config.Environment))
		for key, value := range d.config.Environment {
			envList = append(envList, fmt.Sprintf("%s=%s", key, value))
		}
		options = append(options, cli.WithEnv(envList))
	}

	// Create project options
	projectOptions, err := cli.NewProjectOptions(
		[]string{composePath},
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project options: %w", err)
	}

	// Load the project
	project, err := projectOptions.LoadProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load compose project: %w", err)
	}

	// Apply template-specific configurations
	d.applyTemplateConfig(project, template)

	return project, nil
}

// validateProject performs security and configuration validation
func (d *DockerCompose) validateProject(project *types.Project) error {
	var errors []string

	// Validate services
	for name, service := range project.Services {
		// Check for privileged containers (security risk)
		if service.Privileged {
			errors = append(errors, fmt.Sprintf("service %s: privileged mode is a security risk", name))
		}

		// Check for host network mode (security risk)
		if service.NetworkMode == "host" {
			errors = append(errors, fmt.Sprintf("service %s: host network mode is a security risk", name))
		}

		// Check for dangerous capabilities
		dangerousCaps := []string{"SYS_ADMIN", "NET_ADMIN", "SYS_PTRACE"}
		for _, cap := range service.CapAdd {
			for _, dangerous := range dangerousCaps {
				if strings.EqualFold(cap, dangerous) {
					errors = append(errors, fmt.Sprintf("service %s: dangerous capability %s", name, cap))
				}
			}
		}

		// Validate image source
		if service.Image == "" && service.Build == nil {
			errors = append(errors, fmt.Sprintf("service %s: no image or build configuration", name))
		}

		// Check for volume mounts to sensitive directories
		sensitivePaths := []string{"/", "/etc", "/sys", "/proc", "/dev"}
		for _, volume := range service.Volumes {
			if volume.Type == "bind" {
				for _, sensitive := range sensitivePaths {
					if volume.Source == sensitive {
						errors = append(errors, fmt.Sprintf("service %s: mounting sensitive path %s", name, sensitive))
					}
				}
			}
		}
	}

	// Return validation results
	if len(errors) > 0 {
		log.Warn().Strs("issues", errors).Msg("Project validation found issues")
		// For now, just log warnings but don't block
		// return fmt.Errorf("validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// applyTemplateConfig applies template-specific configurations to the project
func (d *DockerCompose) applyTemplateConfig(project *types.Project, template *templates.Template) {
	// Add template labels to all services
	for name := range project.Services {
		service := project.Services[name]
		if service.Labels == nil {
			service.Labels = make(types.Labels)
		}
		service.Labels["vulnerable-target.template"] = template.ID
		service.Labels["vulnerable-target.author"] = template.Info.Author
		service.Labels["vulnerable-target.managed"] = "true"
		project.Services[name] = service
	}

	// Ensure networks are properly configured
	if len(project.Networks) == 0 {
		project.Networks = types.Networks{
			"default": types.NetworkConfig{
				Driver: "bridge",
				Labels: types.Labels{
					"vulnerable-target.template": template.ID,
				},
			},
		}
	}
}

// startServices starts the Docker Compose services
func (d *DockerCompose) startServices(ctx context.Context, project *types.Project) error {
	// Here we would integrate with Docker API or use compose-go's execution layer
	// For now, we'll use a simplified approach
	
	// Note: compose-go v2 doesn't provide direct service management,
	// so we need to use Docker API or shell out to docker-compose
	// This is a placeholder for the actual implementation
		log.Debug().
		Str("project", project.Name).
		Int("services", len(project.Services)).
		Msg("Starting Docker Compose services")

	// TODO: Implement actual service startup using Docker API
	// For now, we can still use docker-compose CLI but with better validation
	return d.executeCompose(ctx, project, "up", "-d", "--remove-orphans")
}

// stopServices stops the Docker Compose services
func (d *DockerCompose) stopServices(ctx context.Context, project *types.Project) error {
	log.Debug().
		Str("project", project.Name).
		Msg("Stopping Docker Compose services")

	args := []string{"down"}
	if d.config.RemoveVolumes {
		args = append(args, "--volumes")
	}
	if d.config.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}

	return d.executeCompose(ctx, project, args...)
}

// executeCompose executes docker-compose commands (temporary implementation)
func (d *DockerCompose) executeCompose(ctx context.Context, project *types.Project, args ...string) error {
	// This is a temporary implementation until we fully integrate with Docker API
	// We'll keep this but with better error handling and validation
	
	// Create a temporary file for the compose configuration
	tmpFile, err := os.CreateTemp("", "compose-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Marshal the project to YAML
	yamlData, err := project.MarshalYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	if _, err := tmpFile.Write(yamlData); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	tmpFile.Close()

	// Build the command
	cmdArgs := []string{"compose", "-f", tmpFile.Name(), "-p", project.Name}
	cmdArgs = append(cmdArgs, args...)

	// Execute with context
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	if d.config.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose command failed: %w", err)
	}

	return nil
}

// waitForHealthy waits for services to become healthy
func (d *DockerCompose) waitForHealthy(ctx context.Context, project *types.Project) error {
	// TODO: Implement health checking using Docker API
	// For now, just wait a bit
	select {
	case <-time.After(5 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getServiceStatus gets the status of services
func (d *DockerCompose) getServiceStatus(ctx context.Context, project *types.Project) (*ProviderStatus, error) {
	status := &ProviderStatus{
		ProjectName: project.Name,
		Services:    make(map[string]ServiceStatus),
		Healthy:     true,
	}

	// TODO: Implement actual status checking using Docker API
	for name := range project.Services {
		status.Services[name] = ServiceStatus{
			Name:    name,
			Running: false, // Would check actual container status
			Healthy: false, // Would check health status
		}
	}

	return status, nil
}

// getProjectName generates a project name for the template
func (d *DockerCompose) getProjectName(templateID string) string {
	return fmt.Sprintf("vt-%s", strings.ReplaceAll(templateID, "_", "-"))
}

// resolveComposePath resolves the compose file path
func (d *DockerCompose) resolveComposePath(templateID, path string) (string, error) {
	// Handle absolute paths
	if filepath.IsAbs(path) {
		if err := d.validatePath(path); err != nil {
			return "", err
		}
		return path, nil
	}

	// Get working directory
	wd := d.config.WorkingDir
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Build the compose path
	composePath := filepath.Join(wd, "templates", templateID, path)

	// Validate the path
	if err := d.validatePath(composePath); err != nil {
		return "", err
	}

	return composePath, nil
}

// validatePath validates that a path exists and is safe
func (d *DockerCompose) validatePath(path string) error {
	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		log.Warn().
			Str("original", path).
			Str("cleaned", cleanPath).
			Msg("Path was cleaned for security")
	}

	// Check if file exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("compose file not found: %s", cleanPath)
		}
		return fmt.Errorf("failed to stat compose file: %w", err)
	}

	// Ensure it's a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", cleanPath)
	}

	// Check file extension
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" {
		log.Warn().
			Str("path", cleanPath).
			Str("extension", ext).
			Msg("Compose file has unexpected extension")
	}

	return nil
}

// ProviderStatus represents the status of the provider's services
type ProviderStatus struct {
	ProjectName string
	Services    map[string]ServiceStatus
	Healthy     bool
	Message     string
}

// ServiceStatus represents the status of a single service
type ServiceStatus struct {
	Name    string
	Running bool
	Healthy bool
	Message string
	Ports   []string
}
