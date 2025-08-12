package dockercompose

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
)

func TestNewDockerCompose(t *testing.T) {
	dc := NewDockerCompose()
	
	if dc == nil {
		t.Fatal("NewDockerCompose returned nil")
	}
	
	if dc.config == nil {
		t.Fatal("Config is nil")
	}
	
	if dc.config.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout to be 5 minutes, got %v", dc.config.Timeout)
	}
	
	if !dc.config.RemoveVolumes {
		t.Error("Expected RemoveVolumes to be true")
	}
	
	if !dc.config.RemoveOrphans {
		t.Error("Expected RemoveOrphans to be true")
	}
}

func TestNewDockerComposeWithConfig(t *testing.T) {
	config := &Config{
		Timeout:       10 * time.Minute,
		RemoveVolumes: false,
		RemoveOrphans: false,
		Environment: map[string]string{
			"TEST_ENV": "test_value",
		},
		WorkingDir: "/custom/dir",
		Verbose:    true,
	}
	
	dc := NewDockerComposeWithConfig(config)
	
	if dc == nil {
		t.Fatal("NewDockerComposeWithConfig returned nil")
	}
	
	if dc.config.Timeout != 10*time.Minute {
		t.Errorf("Expected timeout to be 10 minutes, got %v", dc.config.Timeout)
	}
	
	if dc.config.RemoveVolumes {
		t.Error("Expected RemoveVolumes to be false")
	}
	
	if dc.config.Environment["TEST_ENV"] != "test_value" {
		t.Error("Environment variable not set correctly")
	}
	
	if dc.config.WorkingDir != "/custom/dir" {
		t.Error("WorkingDir not set correctly")
	}
}

func TestDockerCompose_Name(t *testing.T) {
	dc := NewDockerCompose()
	
	if dc.Name() != "docker-compose" {
		t.Errorf("Expected name to be 'docker-compose', got '%s'", dc.Name())
	}
}

func TestDockerCompose_validateProject(t *testing.T) {
	dc := NewDockerCompose()
	
	tests := []struct {
		name    string
		project *types.Project
		wantErr bool
	}{
		{
			name: "valid project",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "privileged container",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name:       "web",
						Image:      "nginx:latest",
						Privileged: true,
					},
				},
			},
			wantErr: false, // Currently just logs warning
		},
		{
			name: "host network mode",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name:        "web",
						Image:       "nginx:latest",
						NetworkMode: "host",
					},
				},
			},
			wantErr: false, // Currently just logs warning
		},
		{
			name: "dangerous capabilities",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name:   "web",
						Image:  "nginx:latest",
						CapAdd: []string{"SYS_ADMIN", "NET_ADMIN"},
					},
				},
			},
			wantErr: false, // Currently just logs warning
		},
		{
			name: "no image or build",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name: "web",
					},
				},
			},
			wantErr: false, // Currently just logs warning
		},
		{
			name: "sensitive volume mount",
			project: &types.Project{
				Name: "test-project",
				Services: types.Services{
					"web": {
						Name:  "web",
						Image: "nginx:latest",
						Volumes: []types.ServiceVolumeConfig{
							{
								Type:   "bind",
								Source: "/etc",
								Target: "/host-etc",
							},
						},
					},
				},
			},
			wantErr: false, // Currently just logs warning
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dc.validateProject(tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDockerCompose_applyTemplateConfig(t *testing.T) {
	dc := NewDockerCompose()
	
	template := &templates.Template{
		ID: "test-template",
		Info: templates.Info{
			Author: "test-author",
		},
	}
	
	project := &types.Project{
		Name: "test-project",
		Services: types.Services{
			"web": {
				Name: "web",
			},
		},
	}
	
	dc.applyTemplateConfig(project, template)
	
	// Check if labels were added
	webService := project.Services["web"]
	if webService.Labels["vulnerable-target.template"] != "test-template" {
		t.Error("Template label not added")
	}
	if webService.Labels["vulnerable-target.author"] != "test-author" {
		t.Error("Author label not added")
	}
	if webService.Labels["vulnerable-target.managed"] != "true" {
		t.Error("Managed label not added")
	}
	
	// Check if default network was added
	if len(project.Networks) == 0 {
		t.Error("Default network not added")
	}
	if defaultNet, ok := project.Networks["default"]; ok {
		if defaultNet.Driver != "bridge" {
			t.Error("Default network driver is not bridge")
		}
	} else {
		t.Error("Default network not found")
	}
}

func TestDockerCompose_getProjectName(t *testing.T) {
	dc := NewDockerCompose()
	
	tests := []struct {
		templateID string
		expected   string
	}{
		{"test-template", "vt-test-template"},
		{"test_template", "vt-test-template"},
		{"my_awesome_template", "vt-my-awesome-template"},
	}
	
	for _, tt := range tests {
		t.Run(tt.templateID, func(t *testing.T) {
			result := dc.getProjectName(tt.templateID)
			if result != tt.expected {
				t.Errorf("getProjectName(%s) = %s, want %s", tt.templateID, result, tt.expected)
			}
		})
	}
}

func TestDockerCompose_validatePath(t *testing.T) {
	dc := NewDockerCompose()
	
	// Create a temporary test file
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "docker-compose.yaml")
	if err := os.WriteFile(validFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid yaml file",
			path:    validFile,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "non-existent.yaml"),
			wantErr: true,
		},
		{
			name:    "directory instead of file",
			path:    tmpDir,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dc.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDockerCompose_resolveComposePath(t *testing.T) {
	dc := NewDockerCompose()
	
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "templates", "test-template")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	composeFile := filepath.Join(templateDir, "docker-compose.yaml")
	if err := os.WriteFile(composeFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Set working directory for test
	dc.config.WorkingDir = tmpDir
	
	tests := []struct {
		name       string
		templateID string
		path       string
		want       string
		wantErr    bool
	}{
		{
			name:       "relative path",
			templateID: "test-template",
			path:       "docker-compose.yaml",
			want:       composeFile,
			wantErr:    false,
		},
		{
			name:       "absolute path",
			templateID: "test-template",
			path:       composeFile,
			want:       composeFile,
			wantErr:    false,
		},
		{
			name:       "non-existent relative path",
			templateID: "test-template",
			path:       "non-existent.yaml",
			want:       "",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dc.resolveComposePath(tt.templateID, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveComposePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveComposePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDockerCompose_Status(t *testing.T) {
	dc := NewDockerCompose()
	
	// Create a mock template
	template := &templates.Template{
		ID: "test-template",
		Providers: map[string]templates.ProviderConfig{
			"docker-compose": {
				Path: "docker-compose.yaml",
			},
		},
	}
	
	// This test would need a mock Docker environment or actual Docker setup
	// For now, we'll test that the method doesn't panic
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// The method will fail because we don't have a valid compose file,
	// but it shouldn't panic
	_, _ = dc.Status(template)
	
	// Check context cancellation
	select {
	case <-ctx.Done():
		t.Error("Context was cancelled unexpectedly")
	default:
		// Context should still be active
	}
}

func TestProviderStatus(t *testing.T) {
	status := &ProviderStatus{
		ProjectName: "test-project",
		Services: map[string]ServiceStatus{
			"web": {
				Name:    "web",
				Running: true,
				Healthy: true,
				Message: "Service is healthy",
				Ports:   []string{"80:80/tcp"},
			},
			"db": {
				Name:    "db",
				Running: true,
				Healthy: false,
				Message: "Health check pending",
				Ports:   []string{"5432:5432/tcp"},
			},
		},
		Healthy: false,
		Message: "Some services are not healthy",
	}
	
	if status.ProjectName != "test-project" {
		t.Error("ProjectName not set correctly")
	}
	
	if len(status.Services) != 2 {
		t.Error("Services count mismatch")
	}
	
	webStatus, ok := status.Services["web"]
	if !ok {
		t.Error("Web service not found")
	}
	
	if !webStatus.Running || !webStatus.Healthy {
		t.Error("Web service status incorrect")
	}
	
	if len(webStatus.Ports) != 1 || webStatus.Ports[0] != "80:80/tcp" {
		t.Error("Web service ports incorrect")
	}
}

// Integration test - requires Docker to be running
func TestDockerCompose_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Check if Docker is available
	if _, err := NewDockerClient(); err != nil {
		t.Skip("Docker is not available, skipping integration test")
	}
	
	// Create a simple compose file for testing
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "templates", "integration-test")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	composeContent := `
version: '3.8'
services:
  test:
    image: alpine:latest
    command: sleep 30
    labels:
      test: "true"
`
	
	composeFile := filepath.Join(templateDir, "docker-compose.yaml")
	if err := os.WriteFile(composeFile, []byte(composeContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	// Create DockerCompose instance with custom config
	config := &Config{
		Timeout:       30 * time.Second,
		RemoveVolumes: true,
		RemoveOrphans: true,
		WorkingDir:    tmpDir,
	}
	dc := NewDockerComposeWithConfig(config)
	
	// Create a test template
	template := &templates.Template{
		ID: "integration-test",
		Info: templates.Info{
			Name:   "Integration Test",
			Author: "test",
		},
		Providers: map[string]templates.ProviderConfig{
			"docker-compose": {
				Path: "docker-compose.yaml",
			},
		},
	}
	
	// Test Start
	if err := dc.Start(template); err != nil {
		t.Fatalf("Failed to start services: %v", err)
	}
	
	// Give services time to start
	time.Sleep(2 * time.Second)
	
	// Test Status
	status, err := dc.Status(template)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	
	if status.ProjectName == "" {
		t.Error("Status ProjectName is empty")
	}
	
	// Test Stop
	if err := dc.Stop(template); err != nil {
		t.Fatalf("Failed to stop services: %v", err)
	}
}
