package utils

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
)

// Save file if it does not already exist.
func SaveFile(fileName string, fileContents string, relativePath string) error {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
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
		slog.Debug("File Operation", slog.String("Skipping Existing Drawbridge File", fullFilePath))
		return nil
	}

	// Create folder path if it doesn't exist.
	if _, err := os.Stat(relativePath); errors.Is(err, os.ErrNotExist) {
		slog.Debug("File Operation", slog.String("Create File", fullFilePath))

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
	slog.Debug("File Operation", slog.String("Check If File Exists", fullFilePath))

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
	slog.Debug("File Operation", slog.String("Read File", fullFilePath))

	file, err := os.ReadFile(fullFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		return &file
	}
	return nil
}

// Used when we need to add all listening address ips to the certificate authority and server certificate.
func GetDeviceIPs() ([]net.IP, error) {
	var ips []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}

		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// process IP address
			ips = append(ips, ip)
		}
	}
	return ips, nil

}
