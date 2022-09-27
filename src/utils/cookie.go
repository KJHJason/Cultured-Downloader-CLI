package utils

import (
	"time"
	"errors"
	"net/http"
)

func GetCookie(sessionID string, website string) (http.Cookie, error) {
	var domainName, cookieName string
	var sameSite http.SameSite
	if (website == "fantia") {
		domainName = "fantia.jp"
		cookieName = "_session_id"
		sameSite = http.SameSiteLaxMode
	} else if (website == "fanbox") {
		domainName = ".fanbox.cc"
		cookieName = "FANBOXSESSID"
		sameSite = http.SameSiteNoneMode
	} else {
		return http.Cookie{}, errors.New("invalid website")
	}

	if (sessionID == "") {
		return http.Cookie{}, nil
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
	return cookie, nil
}

func VerifyCookie(cookie http.Cookie, website string) (bool, error) {
	// sends a request to the website to verify the cookie
	var websiteURL string
	if (website == "fantia") {
		websiteURL = "https://fantia.jp/"
	} else if (website == "fanbox") {
		websiteURL = "https://www.fanbox.cc/"
	} else {
		return false, errors.New("invalid website")
	}

	if (cookie.Value == "") {
		return false, nil
	}

	resp, err := CallRequest(websiteURL, 5, []http.Cookie{cookie}, "GET", map[string]string{})
	if (err != nil) {
		return false, err
	}

	// check if the cookie is valid
	cookies := resp.Cookies()
	for _, c := range cookies {
		if (c.Name == cookie.Name && c.Value == cookie.Value) {
			return true, nil
		}
	}
	return false, nil
}