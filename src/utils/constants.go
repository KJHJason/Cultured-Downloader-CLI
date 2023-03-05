package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const (
	// Error codes
	DEV_ERROR = iota + 1000
	UNEXPECTED_ERROR
	OS_ERROR
	INPUT_ERROR
	CMD_ERROR
	CONNECTION_ERROR
	DOWNLOAD_ERROR
	RESPONSE_ERROR
	JSON_ERROR
	HTML_ERROR
)

// Returns the user agent based on the user's OS
func GetUserAgent() string {
	userAgent := map[string]string{
		"linux":   "Mozilla/5.0 (X11; Linux x86_64)",
		"darwin":  "Mozilla/5.0 (Macintosh; Intel Mac OS X 12_6)",
		"windows": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
	}
	userAgentOS, ok := userAgent[runtime.GOOS]
	if !ok {
		panic(
			fmt.Sprintf("Error %d: Failed to get user agent OS as your OS, \"%s\", is not supported", OS_ERROR, runtime.GOOS),
		)
	}
	return fmt.Sprintf("%s AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36", userAgentOS)
}

// Returns the path to the application's config directory
func GetAppPath() string {
	appPath, err := os.UserConfigDir()
	if err != nil {
		panic(
			fmt.Sprintf("Error %d, failed to get user's config directory: %s", OS_ERROR, err.Error()),
		)
	}
	return filepath.Join(appPath, "Cultured-Downloader")
}

// Although the variables below are not
// constants, they are not supposed to be changed
var (
	USER_AGENT               = GetUserAgent()
	APP_PATH                 = GetAppPath()
	NUMBER_REGEX             = regexp.MustCompile(`^\d+$`)
	DOWNLOAD_PATH            = GetDefaultDownloadPath()
	ILLEGAL_PATH_CHARS_REGEX = regexp.MustCompile(`[<>:"/\\|?*]`)
	GDRIVE_URL_REGEX         = regexp.MustCompile(
		`https://drive\.google\.com/(?P<type>file/d|drive/(u/\d+/)?folders)/(?P<id>[\w-]+)`,
	)
	GDRIVE_REGEX_ID_INDEX   = GDRIVE_URL_REGEX.SubexpIndex("id")
	GDRIVE_REGEX_TYPE_INDEX = GDRIVE_URL_REGEX.SubexpIndex("type")
	FANTIA_IMAGE_URL_REGEX  = regexp.MustCompile(
		`original_url\":\"(?P<url>/posts/\d+/album_image\?query=[\w%-]*)\"`,
	)
	FANTIA_REGEX_URL_INDEX = FANTIA_IMAGE_URL_REGEX.SubexpIndex("url")

	// For Pixiv Fanbox
	PASSWORD_TEXTS              = []string{"パス", "Pass", "pass", "密码"}
	EXTERNAL_DOWNLOAD_PLATFORMS = []string{"mega", "gigafile", "dropbox", "mediafire"}
)

const (
	VERSION                        = "1.1.0"
	MAX_RETRY_DELAY                = 2.45
	MIN_RETRY_DELAY                = 0.95
	RETRY_COUNTER                  = 4
	MAX_CONCURRENT_DOWNLOADS       = 5
	PIXIV_MAX_CONCURRENT_DOWNLOADS = 3
	MAX_API_CALLS                  = 10
)
