package utils

import (
	"os"
	"fmt"
	"bytes"
	"strings"
	"strconv"
	"encoding/json"
	"path/filepath"
	"github.com/fatih/color"
)

func PathExists(filepath string) bool {
	// checks if a file or directory exists
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}

func CheckIfFileIsEmpty(filepath string) (bool, error) {
	// check if the file exists and has more than 0 bytes
	if PathExists(filepath) {
		file, err := os.Open(filepath)
		if err != nil {
			return true, err
		}
		fileInfo, err := file.Stat()
		if err != nil {
			return true, err
		}

		if fileInfo.Size() > 0 {
			return false, nil
		}
	}
	return true, nil
}

func LogMessageToPath(message, filePath string) {
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

func RemoveIllegalCharsInPath(dirtyPath string) string {
	partiallyCleanedPath := strings.ReplaceAll(
		strings.TrimSpace(dirtyPath),
		".",
		" ",
	)
	return ILLEGAL_PATH_CHARS_REGEX.ReplaceAllString(
		partiallyCleanedPath,
		"_",
	)
}

func CreatePostFolder(downloadPath, creatorName, postId, postTitle string) string {
	creatorName = RemoveIllegalCharsInPath(creatorName)
	postTitle = RemoveIllegalCharsInPath(postTitle)

	postFolderPath := filepath.Join(
		downloadPath,
		creatorName,
		fmt.Sprintf("[%s] %s", postId, postTitle),
	)
	os.MkdirAll(postFolderPath, 0755)
	return postFolderPath
}

type ConfigFile struct {
	DownloadDir string `json:"download_directory"`
	Language string `json:"language"`
	ClientDigestMethod string `json:"client_digest_method"`
}

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

func PretifyJSON(jsonBytes []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, jsonBytes, "", "    ")
	if err != nil {
		return []byte{}, err
	}
	return prettyJSON.Bytes(), nil
}

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