package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// Returns a boolean value indicating whether the specified site supports HTTP/3
//
// Usually, the API endpoints of a site do not support HTTP/3, so the isApi parameter must be provided.
func IsHttp3Supported(site string, isApi bool) bool {
	switch site {
	case FANTIA:
		return !isApi
	case PIXIV_FANBOX:
		return false
	case PIXIV:
		return !isApi
	case PIXIV_MOBILE:
		return true
	case KEMONO, KEMONO_BACKUP:
		return false
	default:
		panic(
			fmt.Errorf(
				"error %d, invalid site, %q in IsHttp3Supported",
				DEV_ERROR,
				site,
			),
		)
	}
}

// Returns the last part of the given URL string
func GetLastPartOfUrl(url string) string {
	removedParams := strings.SplitN(url, "?", 2)
	splittedUrl := strings.Split(removedParams[0], "/")
	return splittedUrl[len(splittedUrl)-1]
}

// Returns the path without the file extension
func RemoveExtFromFilename(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// Converts a map of string back to a string
func ParamsToString(params map[string]string) string {
	paramsStr := ""
	for key, value := range params {
		paramsStr += fmt.Sprintf("%s=%s&", key, url.QueryEscape(value))
	}
	return paramsStr[:len(paramsStr)-1] // remove the last &
}

// Reads and returns the response body in bytes and closes it
func ReadResBody(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"error %d: failed to read response body from %s due to %v",
			RESPONSE_ERROR,
			res.Request.URL.String(),
			err,
		)
	}
	return body, nil
}
