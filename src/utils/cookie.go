package utils

import (
	"time"
	"net/http"
)

func GetCookie(sessionID string, website string) http.Cookie {
	var domainName, cookieName string
	var sameSite http.SameSite
	if website == "fantia" {
		domainName = "fantia.jp"
		cookieName = "_session_id"
		sameSite = http.SameSiteLaxMode
	} else if website == "fanbox" {
		domainName = ".fanbox.cc"
		cookieName = "FANBOXSESSID"
		sameSite = http.SameSiteNoneMode
	} else {
		panic("invalid website")
	}

	if sessionID == "" {
		return http.Cookie{}
	}

	cookie := http.Cookie{
		Name: cookieName,
		Value: sessionID,
		Domain: domainName,
		Expires: time.Now().Add(365 * 24 * time.Hour),
		Path: "/",
		SameSite: sameSite,
		Secure: true,
		HttpOnly: true,
	}
	return cookie
}

func VerifyCookie(cookie http.Cookie, website string) (bool, error) {
	// sends a request to the website to verify the cookie
	var websiteURL string
	if website == "fantia" {
		websiteURL = "https://fantia.jp/mypage/users/plans"
	} else if website == "fanbox" {
		websiteURL = "https://www.fanbox.cc/creators/supporting"
	} else {
		panic("invalid website")
	}

	if cookie.Value == "" {
		return false, nil
	}

	cookies := []http.Cookie{cookie}
	resp, err := CallRequest(websiteURL, 5, cookies, "HEAD", nil, nil, true)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	// check if the cookie is valid
	return resp.Request.URL.String() == websiteURL, nil
}