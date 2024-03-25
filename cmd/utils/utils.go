package utils

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	// Excludes the file name so we can create the necessary parent folder, if any
	folderPathStrings := strings.Split(fullFilePath, "/")
	fullFolderPath := strings.Join(folderPathStrings[:len(folderPathStrings)-1], "/")

	// Verify the file doesn't exist before opening.
	_, err = os.Open(fullFilePath)
	if !errors.Is(err, os.ErrNotExist) {
		slog.Debug("File Operation", slog.String("Skipping Existing Drawbridge File", fullFilePath))
		return nil
	}

	// Create folder path if it doesn't exist.
	if _, err := os.Stat(fullFolderPath); errors.Is(err, os.ErrNotExist) {
		slog.Debug("File Operation", slog.String("Create File", fullFilePath))
		err := os.MkdirAll(fullFolderPath, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating folder on path %s/%s: %s", fullFolderPath, fileName, err)
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

	_, err = f.Write(fileContents)
	if err != nil {
		return fmt.Errorf("error writing file byte contents: %s", err)
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

func UnzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// This must be from how we output the file names during the build process in the ./build/ files.
	// Maybe we should fix this build, rather than here, in the future.
	if strings.Contains(f.Name, "release/") {
		f.Name = strings.Split(f.Name, "release/")[1]
		if !strings.Contains(f.Name, ".zip") {
			f.Name = fmt.Sprintf("%s.zip", f.Name)
		}
	}
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	// Excludes the file name so we can create the necessary parent folder, if any.
	folderPathStrings := strings.Split(filePath, "/")
	fullFolderPath := strings.Join(folderPathStrings[:len(folderPathStrings)-1], "/")
	if err := os.MkdirAll(filepath.Dir(fullFolderPath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

func ZipSource(source, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

// Called when we want to provide a file path for a Drawbrdige resource or config file.
// This is common when we want to use a library which requires a path to our file, but we aren't
func CreateDrawbridgeFilePath(relativePathWithFilename string) string {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	fullFilePath := filepath.Join(execDirPath, relativePathWithFilename)

	return fullFilePath
}

func GetListeningAddress() string {
	listeningAddressBytes := ReadFile("config/listening_address.txt")
	if listeningAddressBytes != nil {
		return strings.TrimSpace(string(*listeningAddressBytes))
	} else {
		return ""
	}
}
