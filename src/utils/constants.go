package utils

import (
	"fmt"
	"os"
	"net/http"
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
	RESPONSE_ERROR
	DOWNLOAD_ERROR
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
			fmt.Errorf(
				"error %d: Failed to get user agent OS as your OS, \"%s\", is not supported", 
				OS_ERROR, 
				runtime.GOOS,
			),
		)
	}
	return fmt.Sprintf("%s AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36", userAgentOS)
}

// Returns the path to the application's config directory
func GetAppPath() string {
	appPath, err := os.UserConfigDir()
	if err != nil {
		panic(
			fmt.Errorf(
				"error %d, failed to get user's config directory: %v", 
				OS_ERROR, 
				err,
			),
		)
	}
	return filepath.Join(appPath, "Cultured-Downloader")
}

const (
	VERSION                        = "1.1.0"
	MAX_RETRY_DELAY                = 2.45
	MIN_RETRY_DELAY                = 0.95
	RETRY_COUNTER                  = 4
	MAX_CONCURRENT_DOWNLOADS       = 5
	PIXIV_MAX_CONCURRENT_DOWNLOADS = 3
	MAX_API_CALLS                  = 10

	FANTIA                    = "fantia"
	FANTIA_TITLE              = "Fantia"
	PIXIV                     = "pixiv"
	PIXIV_TITLE               = "Pixiv"
	PIXIV_FANBOX              = "fanbox"
	PIXIV_FANBOX_TITLE        = "Pixiv Fanbox"

	PASSWORD_FILENAME = "detected_passwords.txt"
	ATTACHMENT_FOLDER = "attachments"
	IMAGES_FOLDER 	  = "images"
	GDRIVE_FOLDER 	  = "gdrive"
)

type cookieInfo struct {
	Domain   string
	Name     string
	SameSite http.SameSite
}

// Although the variables below are not
// constants, they are not supposed to be changed
var (
	USER_AGENT               = GetUserAgent()
	APP_PATH                 = GetAppPath()
	DOWNLOAD_PATH            = GetDefaultDownloadPath()

	SESSION_COOKIE_MAP = map[string]cookieInfo{
		FANTIA: {
			Domain:   "fantia.jp",
			Name:     "_session_id",
			SameSite: http.SameSiteLaxMode,
		},
		PIXIV_FANBOX: {
			Domain:   ".fanbox.cc",
			Name:     "FANBOXSESSID",
			SameSite: http.SameSiteNoneMode,
		},
		PIXIV: {
			Domain:   ".pixiv.net",
			Name:     "PHPSESSID",
			SameSite: http.SameSiteNoneMode,
		},
	}

	NUMBER_REGEX             = regexp.MustCompile(`^\d+$`)
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

	// For readability for the user
	API_TITLE_MAP = map[string]string{
		FANTIA:       FANTIA_TITLE,
		PIXIV_FANBOX: PIXIV_FANBOX_TITLE,
		PIXIV:        PIXIV_TITLE,
	}
)
