package utils

import (
	"os"
	"runtime"
	"path/filepath"
)

func GetUserAgent() string {
	var userAgent = map[string]string {
		"linux":
			"Mozilla/5.0 (X11; Linux x86_64)",
		"darwin":
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 12_6)",
	}
	userAgentOS := userAgent[runtime.GOOS]
	if (userAgentOS == "") {
		userAgentOS = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	}
	return userAgentOS + " AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36"
}

func GetAppPath() string {
	appPath, err := os.UserHomeDir()
	if (err != nil) {
		errorMsg := "failed to get user home directory: " + err.Error()
		panic(errorMsg)
	}
	var appDir = map[string]string {
		"windows": "AppData/Roaming/Cultured-Downloader",
		"linux": ".config/Cultured-Downloader",
		"darwin": "Library/Preferences/Cultured-Downloader",
	}
	appDirOS := appDir[runtime.GOOS]
	if (appDirOS == "") {
		panic("unsupported OS")
	}
	return filepath.Join(appPath, appDirOS)
}

var (
	USER_AGENT = GetUserAgent()
	APP_PATH = GetAppPath()
	DOWNLOAD_PATH = GetDefaultDownloadPath()
)

const (
	RETRY_COUNTER = 5
	MAX_CONCURRENT_DOWNLOADS = 5
)