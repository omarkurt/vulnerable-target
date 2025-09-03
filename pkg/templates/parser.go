package templates

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

var (
	templateIDRegex     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	allowedProviderExts = map[string]bool{
		".yml":  true,
		".yaml": true,
	}
)

func LoadTemplate(filepath string) (Template, error) {
	var template Template
	file, err := os.ReadFile(path.Join(filepath, "index.yaml"))
	if err != nil {
		return template, err
	}
	err = yaml.Unmarshal(file, &template)
	if err != nil {
		return template, err
	}
	err = validateTemplate(template)
	return template, err
}

func validateTemplate(template Template) error {
	if template.ID == "" {
		return fmt.Errorf("template ID is missing")
	}

	if !templateIDRegex.MatchString(template.ID) {
		return fmt.Errorf("template ID '%s' contains invalid characters", template.ID)
	}

	if len(template.Providers) == 0 {
		return fmt.Errorf("no providers specified in the template")
	}

	for name, provider := range template.Providers {
		providerPath := provider.Path

		if providerPath == "" {
			return fmt.Errorf("provider '%s': path is empty", name)
		}

		if filepath.IsAbs(providerPath) {
			return fmt.Errorf("provider '%s': absolute paths are not allowed", name)
		}

		if strings.Contains(providerPath, "..") {
			return fmt.Errorf("provider '%s': path contains invalid '..' segments", name)
		}

		ext := filepath.Ext(providerPath)
		if !isAllowedExtension(ext) {
			return fmt.Errorf("provider '%s': provider file must have one of the allowed extensions: %v", name, allowedProviderExts)
		}
	}

	return nil
}

func isAllowedExtension(ext string) bool {
	return allowedProviderExts[ext]
}
