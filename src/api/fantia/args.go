package fantia

import (
	"fmt"
	"sync"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
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
	utils.ValidateIds(f.PostIds)
	utils.ValidateIds(f.FanclubIds)
	f.PostIds = utils.RemoveSliceDuplicates(f.PostIds)

	if len(f.FanclubPageNums) > 0 {
		utils.ValidatePageNumInput(
			len(f.FanclubIds),
			f.FanclubPageNums,
			[]string{
				"Number of Fantia Fanclub ID(s) and page numbers must be equal.",
			},
		)
	} else {
		f.FanclubPageNums = make([]string, len(f.FanclubIds))
	}

	f.FanclubIds, f.FanclubPageNums = utils.RemoveDuplicateIdAndPageNum(
		f.FanclubIds,
		f.FanclubPageNums,
	)
}

// FantiaDlOptions is the struct that contains the options for downloading from Fantia.
type FantiaDlOptions struct {
	DlThumbnails    bool
	DlImages        bool
	DlAttachments   bool
	DlGdrive        bool

	GdriveClient    *gdrive.GDrive

	SessionCookieId string
	SessionCookies  []*http.Cookie

	csrfMu          sync.Mutex
	CsrfToken       string
}

// GetCsrfToken gets the CSRF token from Fantia's index HTML
// which is required to communicate with their API.
func (f *FantiaDlOptions) GetCsrfToken(userAgent string) error {
	f.csrfMu.Lock()
	defer f.csrfMu.Unlock()

	useHttp3 := utils.IsHttp3Supported(utils.FANTIA, false)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Method:      "GET",
			Url:         "https://fantia.jp/",
			Cookies:     f.SessionCookies,
			Http2:       !useHttp3,
			Http3:       useHttp3,
			CheckStatus: true,
			UserAgent:   userAgent,
		},
	)
	if err != nil {
		err = fmt.Errorf(
			"fantia error %d, failed to get CSRF token from Fantia: %w", 
			utils.CONNECTION_ERROR, 
			err,
		)
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf(
			"fantia error %d, failed to get CSRF token from Fantia: %w", 
			utils.RESPONSE_ERROR, 
			err,
		)
		return err
	}

	// parse the response
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		err = fmt.Errorf(
			"fantia error %d, failed to parse response body when getting CSRF token from Fantia: %w", 
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
			"fantia error %d, failed to get CSRF Token from Fantia, please report this issue!\nHTML: %s",
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
func (f *FantiaDlOptions) ValidateArgs(userAgent string) error {
	if f.SessionCookieId != "" {
		f.SessionCookies = []*http.Cookie{
			api.VerifyAndGetCookie(utils.FANTIA, f.SessionCookieId, userAgent),
		}
	}

	if f.DlGdrive && f.GdriveClient == nil {
		f.DlGdrive = false
	} else if !f.DlGdrive && f.GdriveClient != nil {
		f.GdriveClient = nil
	}

	f.csrfMu = sync.Mutex{}
	return f.GetCsrfToken(userAgent)
}
