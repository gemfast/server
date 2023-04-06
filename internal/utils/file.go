package utils

import (
	"os"
	"errors"
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