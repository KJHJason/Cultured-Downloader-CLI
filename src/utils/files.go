package utils

import (
	"os"
	"fmt"
	"sync"
	"bytes"
	"strings"
	"strconv"
	"encoding/json"
	"path/filepath"
	"github.com/fatih/color"
)

// checks if a file or directory exists
func PathExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}

// Returns the file size based on the provided file path
//
// If the file does not exist or 
// there was an error opening the file at the given file path string, -1 is returned
func GetFileSize(filePath string) (int64, error) {
	if PathExists(filePath) {
		file, err := os.Open(filePath)
		if err != nil {
			return -1, err
		}
		fileInfo, err := file.Stat()
		if err != nil {
			return -1, err
		}
		return fileInfo.Size(), nil
	}
	return -1, nil
}

// Thread-safe logging function that logs to the provided file path
var logToPathMutex sync.Mutex
func LogMessageToPath(message, filePath string) {
	logToPathMutex.Lock()
	defer logToPathMutex.Unlock()

	os.MkdirAll(filepath.Dir(filePath), 0755)
	var err error
	var logFile *os.File
	if !PathExists(filePath) {
		logFile, err = os.Create(filePath)
		if err != nil {
			panic(err)
		}
	} else {
		logFile, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
	}
	defer logFile.Close()

	_, err = logFile.WriteString(message)
	if err != nil {
		panic(err)
	}
} 

// Removes any illegal characters in a path name 
// to prevent any error with file I/O using the path name
func RemoveIllegalCharsInPathName(dirtyPathName string) string {
	dirtyPathName = strings.TrimSpace(dirtyPathName)
	partiallyCleanedPathName := strings.ReplaceAll(dirtyPathName, ".", " ")
	return ILLEGAL_PATH_CHARS_REGEX.ReplaceAllString(partiallyCleanedPathName, "-")
}

// Returns a directory path for a post, artwork, etc. 
// based on the user's saved download path and the provided arguments
func GetPostFolder(downloadPath, creatorName, postId, postTitle string) string {
	creatorName = RemoveIllegalCharsInPathName(creatorName)
	postTitle = RemoveIllegalCharsInPathName(postTitle)

	postFolderPath := filepath.Join(
		downloadPath,
		creatorName,
		fmt.Sprintf("[%s] %s", postId, postTitle),
	)
	return postFolderPath
}

type ConfigFile struct {
	DownloadDir string `json:"download_directory"`
	Language string `json:"language"`
	ClientDigestMethod string `json:"client_digest_method"`
}

// Returns the download path from the config file
func GetDefaultDownloadPath() string {
	configFilePath := filepath.Join(APP_PATH, "config.json")
	if !PathExists(configFilePath) {
		return ""
	}

	configFile, err := os.ReadFile(configFilePath)
	if err != nil {
		os.Remove(configFilePath)
		return ""
	}

	var config ConfigFile
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		os.Remove(configFilePath)
		return ""
	}

	if !PathExists(config.DownloadDir) {
		return ""
	}
	return config.DownloadDir
}

// Pretify a JSON bytes input by indenting it with 4 whitespaces
func PretifyJSON(jsonBytes []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, jsonBytes, "", "    ")
	if err != nil {
		return []byte{}, err
	}
	return prettyJSON.Bytes(), nil
}

// Configure and saves the config file with updated download path
func SetDefaultDownloadPath(newDownloadPath string) {
	if !PathExists(newDownloadPath) {
		color.Red("Download path does not exist. Please create the directory and try again.")
		os.Exit(1)
	}

	os.MkdirAll(APP_PATH, 0755)
	configFilePath := filepath.Join(APP_PATH, "config.json")
	if !PathExists(configFilePath) {
		os.Create(configFilePath)

		is64Bit := strconv.IntSize == 64
		digestMethod := "sha256"
		if (is64Bit) {
			digestMethod = "sha512"
		}
		config := ConfigFile{
			DownloadDir: newDownloadPath,
			Language: "en",
			ClientDigestMethod: digestMethod,
		}

		configFile, err := json.Marshal(config)
		if err != nil {
			panic(err)
		}

		configFile, err = PretifyJSON(configFile)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(configFilePath, configFile, 0644)
		if err != nil {
			panic(err)
		}
	} else {
		// read the file
		configFile, err := os.ReadFile(configFilePath)
		if err != nil {
			panic(err)
		}

		var config ConfigFile
		err = json.Unmarshal(configFile, &config)
		if err != nil {
			panic(err)
		}

		// update the file if the download directory is different
		if (config.DownloadDir == newDownloadPath) {
			return
		}

		config.DownloadDir = newDownloadPath
		configFile, err = json.Marshal(config)
		if err != nil {
			panic(err)
		}

		// indent the file
		configFile, err = PretifyJSON(configFile)
		if err != nil {
			panic(err)
		}

		err = os.WriteFile(configFilePath, configFile, 0644)
		if err != nil {
			panic(err)
		}
	}
}