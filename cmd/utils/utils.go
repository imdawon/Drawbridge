package utils

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
)

// Save file if it does not already exist.
func SaveFile(fileName string, fileContents string, relativePath string) error {
	fullFilePath := fmt.Sprintf("%s/%s", relativePath, fileName)
	// Verify the file doesn't exist before opening.
	_, err := os.Open(fullFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		slog.Info(fmt.Sprintf("Skipping existing Drawbridge file: %s", fullFilePath))
		return nil
	}

	// Create folder path if it doesn't exist.
	if _, err := os.Stat(relativePath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(relativePath, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}

	f, err := os.Create(fullFilePath)
	if err != nil {
		return fmt.Errorf("error creating file: %s/%s", relativePath, fileName)
	}
	defer f.Close()

	_, err = f.WriteString(fileContents)
	if err != nil {
		return fmt.Errorf("error writing file contents: %s", err)
	}

	return nil
}

func FileExists(pathWithFilename string) bool {
	_, err := os.Open(pathWithFilename)
	return !errors.Is(err, os.ErrNotExist)
}

func ReadFile(pathWithFilename string) *[]byte {
	file, err := os.ReadFile(pathWithFilename)
	if !errors.Is(err, os.ErrNotExist) {
		return &file
	}
	return nil
}
