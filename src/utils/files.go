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

// Used in CleanPathName to remove illegal characters in a path name
func removeIllegalRuneInPath(r rune) rune {
	if strings.ContainsRune("<>:\"/\\|?*\n\r\t", r) {
		return '-'
	}
	return r
}

// Removes any illegal characters in a path name
// to prevent any error with file I/O using the path name
func CleanPathName(pathName string) string {
	pathName = strings.TrimSpace(pathName)
	if len(pathName) > 255 {
		pathName = pathName[:255]
	}
	return strings.Map(removeIllegalRuneInPath, pathName)
}

// Returns a directory path for a post, artwork, etc.
// based on the user's saved download path and the provided arguments
func GetPostFolder(downloadPath, creatorName, postId, postTitle string) string {
	creatorName = CleanPathName(creatorName)
	postTitle = CleanPathName(postTitle)

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
