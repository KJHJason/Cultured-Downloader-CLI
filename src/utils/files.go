package utils

import (
	"os"
	"fmt"
	"bytes"
	"strconv"
	"encoding/json"
	"path/filepath"
)

func PathExists(filepath string) bool {
	// checks if a file or directory exists
	_, err := os.Stat(filepath)
	return err == nil
}

func CreateDirectory(filepath string) error {
	// creates a directory
	err := os.MkdirAll(filepath, 0755)
	if (err != nil) {
		return err
	}
	return nil
}

type ConfigFile struct {
	DownloadDir string `json:"download_directory"`
	Language string `json:"language"`
	ClientDigestMethod string `json:"client_digest_method"`
}

func GetDefaultDownloadPath() string {
	configFilePath := filepath.Join(APP_PATH, "config.json")
	if (!PathExists(configFilePath)) {
		return ""
	}

	configFile, err := os.ReadFile(configFilePath)
	if (err != nil) {
		os.Remove(configFilePath)
		return ""
	}

	var config ConfigFile
	err = json.Unmarshal(configFile, &config)
	if (err != nil) {
		os.Remove(configFilePath)
		return ""
	}

	if (!PathExists(config.DownloadDir)) {
		return ""
	}
	return config.DownloadDir
}

func PretifyJSON(jsonBytes []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, jsonBytes, "", "    ")
	if (err != nil) {
		return []byte{}, err
	}
	return prettyJSON.Bytes(), nil
}

func SetDefaultDownloadPath(newDownloadPath string) {
	if (!PathExists(newDownloadPath)) {
		fmt.Println("Download path does not exist. Please create the directory and try again.")
		os.Exit(1)
	}

	CreateDirectory(APP_PATH)
	configFilePath := filepath.Join(APP_PATH, "config.json")
	if (!PathExists(configFilePath)) {
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
		if (err != nil) {
			panic(err)
		}

		configFile, err = PretifyJSON(configFile)
		if (err != nil) {
			panic(err)
		}

		err = os.WriteFile(configFilePath, configFile, 0644)
		if (err != nil) {
			panic(err)
		}
	} else {
		// read the file
		configFile, err := os.ReadFile(configFilePath)
		if (err != nil) {
			panic(err)
		}

		var config ConfigFile
		err = json.Unmarshal(configFile, &config)
		if (err != nil) {
			panic(err)
		}

		// update the file if the download directory is different
		if (config.DownloadDir == newDownloadPath) {
			return
		}

		config.DownloadDir = newDownloadPath
		configFile, err = json.Marshal(config)
		if (err != nil) {
			panic(err)
		}

		// indent the file
		configFile, err = PretifyJSON(configFile)
		if (err != nil) {
			panic(err)
		}

		err = os.WriteFile(configFilePath, configFile, 0644)
		if (err != nil) {
			panic(err)
		}
	}
}