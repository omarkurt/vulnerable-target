package file

import (
	"os"
	"path/filepath"
)

func CreateTempFile(content string, name string) (string, error) {
	dir := filepath.Join(os.TempDir(), "vt-file")

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(dir, name)

	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
