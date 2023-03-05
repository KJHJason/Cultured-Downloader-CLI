package pixivfanbox

import (
	"net/http"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
)

type PixivFanboxDl struct {
	CreatorIds []string
	PostIds    []string
}

type PixivFanboxDlOptions struct {
	DlThumbnails  bool
	DlImages      bool
	DlAttachments bool
	DlGdrive      bool

	SessionCookieId string
	SessionCookies  []http.Cookie
}

func (pf *PixivFanboxDlOptions) ValidateArgs() {
	if pf.SessionCookieId != "" {
		pf.SessionCookies = []http.Cookie{
			api.VerifyAndGetCookie(api.PixivFanbox, api.PixivFanboxTitle, pf.SessionCookieId),
		}
	}
}
