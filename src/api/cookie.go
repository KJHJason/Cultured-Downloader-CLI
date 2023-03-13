package api

import (
	"fmt"
	"net/http"
	"time"
	"os"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

// Returns a cookie with given value and website to be used in requests
func GetCookie(sessionID, website string) *http.Cookie {
	if sessionID == "" {
		return &http.Cookie{}
	}

	var domain, cookieName string
	var sameSite http.SameSite
	if sessionCookieInfo, ok := utils.SESSION_COOKIE_MAP[website]; !ok {
		// Shouldn't happen but could happen during development
		panic(
			fmt.Errorf("error %d, invalid website, \"%s\", in GetCookie", utils.DEV_ERROR, website),
		)
	} else {
		domain = sessionCookieInfo.Domain
		cookieName = sessionCookieInfo.Name
		sameSite = sessionCookieInfo.SameSite
	}

	cookie := http.Cookie{
		Name:     cookieName,
		Value:    sessionID,
		Domain:   domain,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		Path:     "/",
		SameSite: sameSite,
		Secure:   true,
		HttpOnly: true,
	}
	return &cookie
}

// Verifies the given cookie by making a request to the website
// and returns true if the cookie is valid
func VerifyCookie(cookie *http.Cookie, website string) (bool, error) {
	// sends a request to the website to verify the cookie
	var websiteURL string
	var useHttp3 bool
	switch website {
	case utils.FANTIA:
		useHttp3 = true
		websiteURL = utils.FANTIA_URL + "/mypage/users/plans"
	case utils.PIXIV_FANBOX:
		useHttp3 = false
		websiteURL = utils.PIXIV_FANBOX_URL + "/creators/supporting"
	case utils.PIXIV:
		useHttp3 = true
		websiteURL = utils.PIXIV_URL + "/manage/requests"
	default:
		// Shouldn't happen but could happen during development
		panic(
			fmt.Errorf(
				"error %d, invalid website, \"%s\", in VerifyCookie", 
				utils.DEV_ERROR, 
				website,
			),
		)
	}

	if cookie.Value == "" {
		return false, nil
	}

	cookies := []*http.Cookie{cookie}
	resp, err := request.CallRequest(
		&request.RequestArgs{
			Method:      "HEAD",
			Url:         websiteURL,
			Cookies:     cookies,
			CheckStatus: true,
			Http3:       useHttp3,
			Http2:       !useHttp3,
		},
	)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	// check if the cookie is valid
	return resp.Request.URL.String() == websiteURL, nil
}

// Verifies the given cookie by making a request to the website and checks if the cookie is valid
// If the cookie is valid, the cookie will be returned
//
// However, if the cookie is invalid, an error message will be printed out and the program will shutdown
func VerifyAndGetCookie(website, cookieValue string) *http.Cookie {
	if _, ok := utils.API_TITLE_MAP[website]; !ok {
		// Shouldn't happen but could happen during development
		panic(
			fmt.Errorf(
				"error %d, invalid website, \"%s\", in VerifyAndGetCookie", 
				utils.DEV_ERROR, 
				website,
			),
		)
	}

	cookie := GetCookie(cookieValue, website)
	cookieIsValid, err := VerifyCookie(cookie, website)
	if err != nil {
		utils.LogError(err, "Error occurred when trying to verify cookie.", true)
	}
	if cookieValue != "" && !cookieIsValid {
		color.Red(
			fmt.Sprintf(
				"error %d: %s cookie is invalid", 
				utils.INPUT_ERROR, 
				utils.API_TITLE_MAP[website],
			),
		)
		os.Exit(1)
	}
	return cookie
}
