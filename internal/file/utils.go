package file

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateTempFile(content string, name string) (string, error) {
	tempDir := filepath.Join(os.TempDir(), "vt-folder")

	err := os.MkdirAll(tempDir, 0700)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	filePath := filepath.Join(tempDir, name)

	err = os.WriteFile(filePath, []byte(content), 0600)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func DeteleFile(path string) error {
	return os.RemoveAll(path)
}
