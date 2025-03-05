package utils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Save file if it does not already exist.
func SaveFile(fileName string, fileContents string, relativePath string) error {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
	}
	execDirPath := path.Dir(execPath)
	relativePath = filepath.Join(execDirPath, relativePath)
	fullFilePath := filepath.Join(relativePath, fileName)
	// Excludes the file name so we can create the necessary parent folder, if any

	// Verify the file doesn't exist before opening.
	_, err = os.Open(fullFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		slog.Debug("File Operation", slog.String("Skipping Existing Drawbridge File", fullFilePath))
		return nil
	}

	// Create folder path if it doesn't exist.
	if _, err := os.Stat(relativePath); errors.Is(err, os.ErrNotExist) {
		slog.Debug("File Operation", slog.String("Create Folder", relativePath))
		err := os.MkdirAll(relativePath, os.ModePerm)
		if err != nil {
			slog.Error("File Operation", slog.Any("Fail - Creating Folder Path", err))
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

// TODO
// Remove this and refactor SaveFile to be generic.
func SaveFileByte(fileName string, fileContents []byte, relativePath string) error {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
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
			slog.Error("Error creating file on path %s/%s: %s", relativePath, fileName, err)
		}
	}

	f, err := os.Create(fullFilePath)
	if err != nil {
		return fmt.Errorf("error creating file: %s/%s", relativePath, fileName)
	}
	defer f.Close()

	_, err = f.Write(fileContents)
	if err != nil {
		return fmt.Errorf("error writing file byte contents: %s", err)
	}

	return nil
}

func DeleteDirectory(relativePath string) error {
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
	}
	execDirPath := path.Dir(execPath)
	fullPath := filepath.Join(execDirPath, relativePath)

	err = os.RemoveAll(fullPath) // delete an entire directory
	if err != nil {
		slog.Error("File Operation", slog.Any("Fail - Deleting Drawbridge File/Folder", fullPath))
		return err
	}
	return nil
}

func FileExists(pathWithFilename string) bool {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
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
		slog.Error(err.Error())
		return nil
	}
	execDirPath := path.Dir(execPath)
	fullFilePath := filepath.Join(execDirPath, pathWithFilename)
	slog.Debug("File Operation", slog.String("Read File", fullFilePath))

	file, err := os.ReadFile(fullFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Error("Error reading file", slog.String("path", fullFilePath), slog.Any("error", err))
		}
		return nil
	}
	return &file
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

func CopyFile(filePath string, destinationPath string) error {
	fileBytes := ReadFile(filePath)
	if fileBytes == nil {
		return fmt.Errorf("error reading source file: %s", filePath)
	}
	
	// Use filepath functions instead of manual string splitting for safety
	filename := filepath.Base(filePath)

	err := SaveFileByte(filename, *fileBytes, destinationPath)
	if err != nil {
		return err
	}
	return nil
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(srcZipFile, destDir string) error {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
	}
	execDirPath := path.Dir(execPath)
	srcZipFile = filepath.Join(execDirPath, srcZipFile)
	destDir = filepath.Join(execDirPath, destDir)

	// Open the ZIP file
	reader, err := zip.OpenReader(srcZipFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Iterate through the files in the ZIP archive
	slog.Debug("iterating over zip files to unzip...")
	for _, file := range reader.File {
		// Get the file path inside the ZIP archive
		filePath := filepath.Join(destDir, file.Name)

		// Create the parent directories if they don't exist
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Create the file on the filesystem
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		// Open a reader for the file inside the ZIP archive
		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		defer rc.Close()

		// Copy the contents of the file from the ZIP archive to the filesystem
		_, err = io.Copy(outFile, rc)
		if err != nil {
			outFile.Close()
			return err
		}

		outFile.Close()
	}

	return nil
}

func ZipSource(srcDir, destZipFile string) error {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
	}
	execDirPath := path.Dir(execPath)
	srcDir = filepath.Join(execDirPath, srcDir)
	destZipFile = filepath.Join(execDirPath, destZipFile)

	// Create a new ZIP file
	zipFile, err := os.Create(destZipFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// Create a new ZIP writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through the source directory and add files to the ZIP
	return filepath.Walk(srcDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path of the file from the source directory
		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create a new file header for the file
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Update the file header with the relative path
		// Use forward slashes and set the flags for better cross-platform compatibility
		header.Name = filepath.ToSlash(relPath)
		header.Method = zip.Deflate

		// Create a new file writer inside the ZIP file
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// If it's a directory, skip writing any content
		if info.IsDir() {
			return nil
		}

		// Open the source file and copy its contents to the ZIP file
		srcFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(writer, srcFile)
		return err
	})
}

// Called when we want to provide a file path for a Drawbrdige resource or config file.
// This is common when we want to use a library which requires a path to our file, but we aren't
func CreateDrawbridgeFilePath(relativePathWithFilename string) string {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		slog.Error(err.Error())
	}
	execDirPath := path.Dir(execPath)
	fullFilePath := filepath.Join(execDirPath, relativePathWithFilename)

	return fullFilePath
}

func PadWithZeros(num int) string {
	return fmt.Sprintf("%03d", num)
}

func NewUUID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return uuid, nil
}

func RandInt(min, max int) int {
	return min + rand.Intn(max-min)
}

func BeautifulTimeSince(timestamp string) string {
	// Parse the timestamp string
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "Invalid timestamp"
	}

	// Calculate the duration between the timestamp and the current time
	duration := time.Since(t)

	// Define the time thresholds
	minuteThreshold := time.Minute
	hourThreshold := time.Hour
	dayThreshold := 24 * time.Hour

	switch {
	case duration < minuteThreshold:
		return "Less than a minute ago"
	case duration < 2*minuteThreshold:
		return "Around a minute ago"
	case duration < hourThreshold:
		minutes := int(duration.Minutes())
		return fmt.Sprintf("Around %d minutes ago", minutes)
	case duration < 2*hourThreshold:
		return "Around 1 hour ago"
	case duration < dayThreshold:
		hours := int(duration.Hours())
		return fmt.Sprintf("Around %d hours ago", hours)
	default:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "Around 1 day ago"
		}
		return fmt.Sprintf("Around %d days ago", days)
	}
}

// generatePlaceholders generates a string with n number of SQLite placeholders separated by commas.
func GeneratePlaceholders(n int) string {
	placeholders := make([]string, n)
	for i := 0; i < n; i++ {
		placeholders[i] = "?"
	}
	return strings.Join(placeholders, ", ")
}
