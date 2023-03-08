package pixivfanbox

import (
	"net/http"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// PixivFanboxDl is the struct that contains the IDs of the Pixiv Fanbox creators and posts to download.
type PixivFanboxDl struct {
	CreatorIds      []string
	CreatorPageNums []string

	PostIds    []string
}

// ValidateArgs validates the IDs of the Pixiv Fanbox creators and posts to download.
//
// It also validates the page numbers of the creators to download.
func (pf *PixivFanboxDl) ValidateArgs() {
	utils.ValidateIds(&pf.CreatorIds)
	utils.ValidateIds(&pf.PostIds)

	if len(pf.CreatorPageNums) > 0 {
		utils.ValidatePageNumInput(
			len(pf.CreatorIds),
			&pf.CreatorPageNums,
			[]string{
				"Number of Pixiv Fanbox Creator ID(s) and page numbers must be equal.",
			},
		)
	} else {
		pf.CreatorPageNums = make([]string, len(pf.CreatorIds))
	}
}

// PixivFanboxDlOptions is the struct that contains the options for downloading from Pixiv Fanbox.
type PixivFanboxDlOptions struct {
	DlThumbnails  bool
	DlImages      bool
	DlAttachments bool
	DlGdrive      bool

	SessionCookieId string
	SessionCookies  []http.Cookie
}

// ValidateArgs validates the session cookie ID of the Pixiv Fanbox account to download from.
func (pf *PixivFanboxDlOptions) ValidateArgs() {
	if pf.SessionCookieId != "" {
		pf.SessionCookies = []http.Cookie{
			api.VerifyAndGetCookie(utils.PIXIV_FANBOX, pf.SessionCookieId),
		}
	}
}
