package fantia

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
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

func (f *FantiaDlOptions) GetCsrfToken() error {
	res, err := request.CallRequest(
		"GET", 
		"https://fantia.jp/", 
		30, 
		f.SessionCookies, 
		nil, 
		nil, 
		false,
	)
	if err != nil || res.StatusCode != 200 {
		err = fmt.Errorf("error %d, failed to get CSRF token from Fantia: %w", utils.CONNECTION_ERROR, err)
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf("error %d, failed to get CSRF token from Fantia: %w", utils.RESPONSE_ERROR, err)
		return err
	}

	// parse the response
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		err = fmt.Errorf("error %d, failed to parse response body when getting CSRF token from Fantia: %w", utils.HTML_ERROR, err)
		return err
	}

	if csrfToken, ok := doc.Find("meta[name=csrf-token]").Attr("content"); !ok {
		// shouldn't happen but just in case if Fantia's csrf token changes
		docHtml, err := doc.Html()
		if err != nil {
			docHtml = "failed to get HTML"
		}
		return fmt.Errorf(
			"error %d, failed to get CSRF Token from Fantia, please report this issue!\nHTML: %s",
			utils.HTML_ERROR,
			docHtml,
		)
	} else {
		f.CsrfToken = csrfToken
	}
	return nil
} 

func (f *FantiaDlOptions) ValidateArgs() error {
	if f.SessionCookieId != "" {
		f.SessionCookies = []http.Cookie{
			api.VerifyAndGetCookie(api.Fantia, api.FantiaTitle, f.SessionCookieId),
		}
	}

	return f.GetCsrfToken()
}