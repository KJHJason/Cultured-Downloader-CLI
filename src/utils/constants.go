package utils

import (
	"os"
	"regexp"
	"runtime"
	"path/filepath"
)

func GetUserAgent() string {
	userAgent := map[string]string {
		"linux":
			"Mozilla/5.0 (X11; Linux x86_64)",
		"darwin":
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 12_6)",
	}
	userAgentOS := userAgent[runtime.GOOS]
	if userAgentOS == "" {
		userAgentOS = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	}
	return userAgentOS + " AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36"
}

func GetAppPath() string {
	appPath, err := os.UserHomeDir()
	if err != nil {
		errorMsg := "failed to get user home directory: " + err.Error()
		panic(errorMsg)
	}
	appDir := map[string]string {
		"windows": "AppData/Roaming/Cultured-Downloader",
		"linux": ".config/Cultured-Downloader",
		"darwin": "Library/Preferences/Cultured-Downloader",
	}
	appDirOS := appDir[runtime.GOOS]
	if appDirOS == "" {
		panic("unsupported OS")
	}
	return filepath.Join(appPath, appDirOS)
}

var (
	USER_AGENT = GetUserAgent()
	APP_PATH = GetAppPath()
	PASSWORD_TEXTS = []string {"パス", "Pass", "pass", "密码"}
	EXTERNAL_DOWNLOAD_PLATFORMS = []string {"mega", "gigafile", "dropbox", "mediafire"}
	DOWNLOAD_PATH = GetDefaultDownloadPath()
	UGOIRA_ACCEPTED_EXT = []string {".gif", ".mp4", ".webm"}
	ILLEGAL_PATH_CHARS_REGEX = regexp.MustCompile(`[<>:"/\\|?*]`)
	GDRIVE_URL_REGEX = regexp.MustCompile(
		`https://drive\.google\.com/(?P<type>file/d|drive/(u/\d+/)?folders)/(?P<id>[\w-]+)`,
	)
	GDRIVE_REGEX_ID_INDEX = GDRIVE_URL_REGEX.SubexpIndex("id")
	GDRIVE_REGEX_TYPE_INDEX = GDRIVE_URL_REGEX.SubexpIndex("type")
)

const (
	MAX_RETRY_DELAY = 2.45
	MIN_RETRY_DELAY = 0.95
	RETRY_COUNTER = 5
	MAX_CONCURRENT_DOWNLOADS = 5
	PIXIV_MAX_CONCURRENT_DOWNLOADS = 2
	MAX_API_CALLS = 10
)