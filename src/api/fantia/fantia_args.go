package fantia

import (
	"fmt"
	"sync"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// FantiaDl is the struct that contains the 
// IDs of the Fantia fanclubs and posts to download.
type FantiaDl struct {
	FanclubIds []string
	FanclubPageNums []string

	PostIds    []string
}

// ValidateArgs validates the IDs of the Fantia fanclubs and posts to download.
//
// It also validates the page numbers of the fanclubs to download.
//
// Should be called after initialising the struct.
func (f *FantiaDl) ValidateArgs() {
	utils.ValidateIds(&f.FanclubIds)
	utils.ValidateIds(&f.PostIds)

	if len(f.FanclubPageNums) > 0 {
		utils.ValidatePageNumInput(
			len(f.FanclubIds),
			&f.FanclubPageNums,
			[]string{
				"Number of Fantia Fanclub ID(s) and page numbers must be equal.",
			},
		)
	} else {
		f.FanclubPageNums = make([]string, len(f.FanclubIds))
	}
}

// FantiaDlOptions is the struct that contains the options for downloading from Fantia.
type FantiaDlOptions struct {
	DlThumbnails    bool
	DlImages        bool
	DlAttachments   bool

	SessionCookieId string
	SessionCookies  []http.Cookie

	csrfMu          sync.Mutex
	CsrfToken       string
}

// GetCsrfToken gets the CSRF token from Fantia's index HTML
// which is required to communicate with their API.
func (f *FantiaDlOptions) GetCsrfToken() error {
	f.csrfMu.Lock()
	defer f.csrfMu.Unlock()

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
		err = fmt.Errorf(
			"error %d, failed to get CSRF token from Fantia: %w", 
			utils.CONNECTION_ERROR, 
			err,
		)
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf(
			"error %d, failed to get CSRF token from Fantia: %w", 
			utils.RESPONSE_ERROR, 
			err,
		)
		return err
	}

	// parse the response
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		err = fmt.Errorf(
			"error %d, failed to parse response body when getting CSRF token from Fantia: %w", 
			utils.HTML_ERROR, 
			err,
		)
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

// ValidateArgs validates the options for downloading from Fantia.
//
// Should be called after initialising the struct.
func (f *FantiaDlOptions) ValidateArgs() error {
	if f.SessionCookieId != "" {
		f.SessionCookies = []http.Cookie{
			api.VerifyAndGetCookie(utils.FANTIA, f.SessionCookieId),
		}
	}

	f.csrfMu = sync.Mutex{}
	return f.GetCsrfToken()
}
