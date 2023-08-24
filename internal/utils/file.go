package utils

import (
	"errors"
	"fmt"
	"os"
)

func FileExists(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func RemoveFileIfExists(filePath string) error {
	exists, err := FileExists(filePath)
	if err != nil {
		return err
	} else if exists {
		err := os.Remove(filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func MkDirs(dir string) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create dir %s: %w", dir, err)
	}
	return nil
}
