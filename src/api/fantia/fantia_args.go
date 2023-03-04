package fantia

import (
	"net/http"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
)

type FantiaDl struct {
	FanclubIds []string
	PostIds    []string
}

type FantiaDlOptions struct {
	DlThumbnails    bool
	DlImages        bool
	DlAttachments   bool

	SessionCookieId string
	SessionCookies  []http.Cookie

	CsrfToken       string
}

func (f *FantiaDlOptions) GetCsrfToken() {
	// wip
} 

func (f *FantiaDlOptions) ValidateArgs() {
	if f.SessionCookieId != "" {
		f.SessionCookies = []http.Cookie{
			api.VerifyAndGetCookie(api.Fantia, api.FantiaTitle, f.SessionCookieId),
		}
	}

	f.GetCsrfToken()
}