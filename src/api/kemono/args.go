package kemono

import (
	"fmt"
	"os"
	"net/http"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

const (
	BASE_REGEX_STR = `https://kemono\.party/(?P<service>patreon|fanbox|gumroad|subscribestar|dlsite|fantia|boosty)/user/(?P<creatorId>[\w-]+)`
	API_MAX_CONCURRENT = 3
)
var (
	POST_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%s/post/(?P<postId>\d+)$`,
			BASE_REGEX_STR,
		),
	)
	CREATOR_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%s$`,
			BASE_REGEX_STR,
		),
	)
)

type KemonoDl struct {
	CreatorUrls     []string
	CreatorPageNums []string

	PostUrls []string
}

func (k *KemonoDl) ValidateArgs() {
	utils.ValidatePageNumInput(
		len(k.CreatorUrls),
		k.CreatorPageNums,
		[]string{
			"Number of creator URL(s) and page numbers must be equal.",
		},
	)

	valid, outlier := utils.SliceMatchesRegex(CREATOR_URL_REGEX, k.CreatorUrls)
	if !valid {
		color.Red(
			fmt.Sprintf(
				"kemono error %d: invalid creator URL found for kemono party: %s",
				utils.INPUT_ERROR,
				outlier,
			),
		)
		os.Exit(1)
	}

	valid, outlier = utils.SliceMatchesRegex(POST_URL_REGEX, k.PostUrls)
	if !valid {
		color.Red(
			fmt.Sprintf(
				"kemono error %d: invalid post URL found for kemono party: %s",
				utils.INPUT_ERROR,
				outlier,
			),
		)
		os.Exit(1)
	}
}

// KemonoDlOptions is the struct that contains the arguments for Kemono download options.
type KemonoDlOptions struct {
	DlAttachments bool
	DlGdrive      bool

	// GDriveClient is the Google Drive client to be 
	// used in the download process for Pixiv Fanbox posts
	GDriveClient   *gdrive.GDrive

	SessionCookieId string
	SessionCookies  []*http.Cookie
}

// ValidateArgs validates the session cookie ID of the Kemono account to download from.
// It also validates the Google Drive client if the user wants to download to Google Drive.
//
// Should be called after initialising the struct.
func (k *KemonoDlOptions) ValidateArgs(userAgent string) {
	if k.SessionCookieId != "" {
		k.SessionCookies = []*http.Cookie{
			api.VerifyAndGetCookie(utils.KEMONO, k.SessionCookieId, userAgent),
		}
	} else {
		color.Red("kemono error %d: session cookie ID is required", utils.INPUT_ERROR)
	}

	if k.DlGdrive && k.GDriveClient == nil {
		k.DlGdrive = false
	}
}
