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

func DeleteDirectory(relativePath string) error {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
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

func CopyFile(filePath string, destinationPath string) error {
	fileBytes := ReadFile(filePath)
	splitPath := strings.Split(filePath, "/")
	filePathFolderLevels := len(splitPath)
	filename := splitPath[filePathFolderLevels-1:][0]

	err := SaveFileByte(filename, *fileBytes, destinationPath)
	if err != nil {
		return err
	}
	return nil
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(source string, target string) ([]string, error) {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	source = filepath.Join(execDirPath, source)
	target = filepath.Join(execDirPath, target)

	var filenames []string

	r, err := zip.OpenReader(source)
	if err != nil {
		slog.Error("Unzip File", slog.Any("Error", err))
		return filenames, fmt.Errorf("opening zip file failed: %w", err)
	}
	defer r.Close()

	slog.Debug("iterating over zip files to unzip...")
	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(target, f.Name)

		// Check for ZipSlip.
		if !strings.HasPrefix(fpath, path.Join(filepath.Clean(target), string(os.PathSeparator))) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, fmt.Errorf("unzip mkdirall failed: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, fmt.Errorf("unzip open destination filename failed: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, fmt.Errorf("open zip file failed: %w", err)
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, fmt.Errorf("io copy failed: %w", err)
		}
	}
	return filenames, nil
}

func ZipSource(source, target string) error {
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	source = filepath.Join(execDirPath, source)
	target = filepath.Join(execDirPath, target)

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
