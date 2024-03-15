package utils

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

// Save file if it does not already exist.
func SaveFile(fileName string, fileContents string, relativePath string) error {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	relativePath = filepath.Join(execDirPath, relativePath)
	fullFilePath := filepath.Join(relativePath, fileName)

	// Verify the file doesn't exist before opening.
	_, err = os.Open(fullFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		slog.Debug(fmt.Sprintf("SAVE FILE: Skipping existing Drawbridge file: %s", fullFilePath))
		return nil
	}

	// Create folder path if it doesn't exist.
	if _, err := os.Stat(relativePath); errors.Is(err, os.ErrNotExist) {
		slog.Debug("CREATING FOLDER PATH EXIST?: %s", fullFilePath)

		err := os.Mkdir(relativePath, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating file on path %s/%s: %s", relativePath, fileName, err)
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
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	fullFilePath := filepath.Join(execDirPath, pathWithFilename)
	slog.Debug("FILE EXIST?: %s", fullFilePath)

	_, err = os.Open(fullFilePath)
	return !errors.Is(err, os.ErrNotExist)
}

func ReadFile(pathWithFilename string) *[]byte {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	fullFilePath := filepath.Join(execDirPath, pathWithFilename)
	slog.Debug("READING FILE: %s", fullFilePath)

	file, err := os.ReadFile(fullFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		return &file
	}
	return nil
}
