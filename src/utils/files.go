package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if !PathExists(filePath) {
		return -1, os.ErrNotExist
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		return -1, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return -1, err
	}
	return fileInfo.Size(), nil
}

// Thread-safe logging function that logs to the provided file path
func LogMessageToPath(message, filePath string, level int) {
	os.MkdirAll(filepath.Dir(filePath), 0666)
	logFileContents, err := os.ReadFile(filePath)
	if err != nil {
		err = fmt.Errorf(
			"error %d: failed to read log file, more info => %v\nfile path: %s\noriginal message: %s",
			OS_ERROR,
			err,
			filePath,
			message,
		)
		LogError(err, "", false, ERROR)
		return
	}

	// check if the same message has already been logged
	if strings.Contains(string(logFileContents), message) {
		return
	}

	logFile, err := os.OpenFile(
		filePath,
		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0666,
	)
	if err != nil {
		err = fmt.Errorf(
			"error %d: failed to open log file, more info => %v\nfile path: %s\noriginal message: %s",
			OS_ERROR,
			err,
			filePath,
			message,
		)
		LogError(err, "", false, ERROR)
		return
	}
	defer logFile.Close()

	pathLogger := NewLogger(logFile)
	pathLogger.LogBasedOnLvl(level, message)
}

// Uses bufio.Reader to read a line from a file and returns it as a byte slice
//
// Mostly thanks to https://devmarkpro.com/working-big-files-golang
func ReadLine(reader *bufio.Reader) ([]byte, error) {
	var err error
	var isPrefix = true
	var totalLine, line []byte

	// Read until isPrefix is false as
	// that means the line has been fully read
	for isPrefix && err == nil {
		line, isPrefix, err = reader.ReadLine()
		totalLine = append(totalLine, line...)
	}
	return totalLine, err
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
	Language    string `json:"language"`
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

// saves the new download path to the config file if it does not exist
func saveConfig(newDownloadPath, configFilePath string) error {
	config := ConfigFile{
		DownloadDir: newDownloadPath,
		Language:    "en",
	}
	configFile, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to marshal config file, more info => %v",
			JSON_ERROR,
			err,
		)
	}

	err = os.WriteFile(configFilePath, configFile, 0666)
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to write config file, more info => %v",
			OS_ERROR,
			err,
		)
	}
	return nil
}

// saves the new download path to the config file and overwrites the old one
func overwriteConfig(newDownloadPath, configFilePath string) error {
	// read the file
	configFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to read config file, more info => %v",
			OS_ERROR,
			err,
		)
	}

	var config ConfigFile
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to unmarshal config file, more info => %v",
			JSON_ERROR,
			err,
		)
	}

	// update the file if the download directory is different
	if config.DownloadDir == newDownloadPath {
		return nil
	}

	config.DownloadDir = newDownloadPath
	configFile, err = json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to marshal config file, more info => %v",
			JSON_ERROR,
			err,
		)
	}

	err = os.WriteFile(configFilePath, configFile, 0666)
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to write config file, more info => %v",
			OS_ERROR,
			err,
		)
	}
	return nil
}

// Configure and saves the config file with updated download path
func SetDefaultDownloadPath(newDownloadPath string) error {
	if !PathExists(newDownloadPath) {
		return fmt.Errorf("error %d: download path does not exist, please create the directory and try again", INPUT_ERROR)
	}

	os.MkdirAll(APP_PATH, 0666)
	configFilePath := filepath.Join(APP_PATH, "config.json")
	if !PathExists(configFilePath) {
		return saveConfig(newDownloadPath, configFilePath)
	}
	return overwriteConfig(newDownloadPath, configFilePath)
}
