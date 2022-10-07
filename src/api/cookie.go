package api

import (
	"fmt"
	"net/http"
	"time"
	"os"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/fatih/color"
)

// Returns a cookie with given value and website to be used in requests
func GetCookie(sessionID, website string) http.Cookie {
	var domainName, cookieName string
	var sameSite http.SameSite
	switch website {
	case Fantia:
		domainName = "fantia.jp"
		cookieName = "_session_id"
		sameSite = http.SameSiteLaxMode
	case PixivFanbox:
		domainName = ".fanbox.cc"
		cookieName = "FANBOXSESSID"
		sameSite = http.SameSiteNoneMode
	case Pixiv:
		domainName = ".pixiv.net"
		cookieName = "PHPSESSID"
		sameSite = http.SameSiteNoneMode
	default:
		panic("invalid website")
	}

	if sessionID == "" {
		return http.Cookie{}
	}

	cookie := http.Cookie{
		Name:     cookieName,
		Value:    sessionID,
		Domain:   domainName,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		Path:     "/",
		SameSite: sameSite,
		Secure:   true,
		HttpOnly: true,
	}
	return cookie
}

// Verifies the given cookie by making a request to the website
// and returns true if the cookie is valid
func VerifyCookie(cookie http.Cookie, website string) (bool, error) {
	// sends a request to the website to verify the cookie
	var websiteURL string
	switch website {
	case Fantia:
		websiteURL = "https://fantia.jp/mypage/users/plans"
	case PixivFanbox:
		websiteURL = "https://www.fanbox.cc/creators/supporting"
	case Pixiv:
		websiteURL = "https://www.pixiv.net/manage/requests"
	default:
		panic("invalid website")
	}

	if cookie.Value == "" {
		return false, nil
	}

	cookies := []http.Cookie{cookie}
	resp, err := request.CallRequest("HEAD", websiteURL, 5, cookies, nil, nil, true)
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
func VerifyAndGetCookie(website, title, cookieValue string) http.Cookie {
	if title == "" {
		title = website
	}

	cookie := GetCookie(cookieValue, website)
	cookieIsValid, err := VerifyCookie(cookie, website)
	if err != nil {
		utils.LogError(err, "Error occurred when trying to verify cookie.", true)
	}
	if cookieValue != "" && !cookieIsValid {
		color.Red(fmt.Sprintf("%s cookie is invalid", title))
		os.Exit(1)
	}
	return cookie
}