package pixivfanbox

import (
	"net/http"
	"os"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

// PixivFanboxDl is the struct that contains the IDs of the Pixiv Fanbox creators and posts to download.
type PixivFanboxDl struct {
	CreatorIds      []string
	CreatorPageNums []string

	PostIds []string
}

var creatorIdRegex = regexp.MustCompile(`^[\w.-]+$`)

// ValidateArgs validates the IDs of the Pixiv Fanbox creators and posts to download.
//
// It also validates the page numbers of the creators to download.
//
// Should be called after initialising the struct.
func (pf *PixivFanboxDl) ValidateArgs() {
	utils.ValidateIds(pf.PostIds)
	pf.PostIds = utils.RemoveSliceDuplicates(pf.PostIds)

	for _, creatorId := range pf.CreatorIds {
		if !creatorIdRegex.MatchString(creatorId) {
			color.Red(
				"error %d: invalid Pixiv Fanbox creator ID %q, must be alphanumeric with underscores, dashes, or periods",
				utils.INPUT_ERROR,
				creatorId,
			)
			os.Exit(1)
		}
	}

	if len(pf.CreatorPageNums) > 0 {
		utils.ValidatePageNumInput(
			len(pf.CreatorIds),
			pf.CreatorPageNums,
			[]string{
				"Number of Pixiv Fanbox Creator ID(s) and page numbers must be equal.",
			},
		)
	} else {
		pf.CreatorPageNums = make([]string, len(pf.CreatorIds))
	}
	pf.CreatorIds, pf.CreatorPageNums = utils.RemoveDuplicateIdAndPageNum(
		pf.CreatorIds,
		pf.CreatorPageNums,
	)
}

// PixivFanboxDlOptions is the struct that contains the options for downloading from Pixiv Fanbox.
type PixivFanboxDlOptions struct {
	DlThumbnails  bool
	DlImages      bool
	DlAttachments bool
	DlGdrive      bool

	// GdriveClient is the Google Drive client to be
	// used in the download process for Pixiv Fanbox posts
	GdriveClient *gdrive.GDrive

	SessionCookieId string
	SessionCookies  []*http.Cookie
}

// ValidateArgs validates the session cookie ID of the Pixiv Fanbox account to download from.
//
// Should be called after initialising the struct.
func (pf *PixivFanboxDlOptions) ValidateArgs(userAgent string) {
	if pf.SessionCookieId != "" {
		pf.SessionCookies = []*http.Cookie{
			api.VerifyAndGetCookie(utils.PIXIV_FANBOX, pf.SessionCookieId, userAgent),
		}
	}

	if pf.DlGdrive && pf.GdriveClient == nil {
		pf.DlGdrive = false
	} else if !pf.DlGdrive && pf.GdriveClient != nil {
		pf.GdriveClient = nil
	}
}
