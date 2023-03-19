package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
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
	case KEMONO:
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
		err = fmt.Errorf(
			"error %d: failed to read response body from %s due to %v",
			RESPONSE_ERROR,
			res.Request.URL.String(),
			err,
		)
		return nil, err
	}
	return body, nil
}

func logJsonResponse(body []byte) {
	var prettyJson bytes.Buffer
	err := json.Indent(&prettyJson, body, "", "    ")
	if err != nil {
		color.Red(
			fmt.Sprintf(
				"error %d: failed to indent JSON response body due to %v",
				JSON_ERROR,
				err,
			),
		)
		return
	}

	filename := fmt.Sprintf("saved_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	filePath := filepath.Join("json", filename)
	os.MkdirAll(filePath, 0666)
	err = os.WriteFile(filePath, prettyJson.Bytes(), 0666)
	if err != nil {
		color.Red(
			fmt.Sprintf(
				"error %d: failed to write JSON response body to file due to %v",
				UNEXPECTED_ERROR,
				err,
			),
		)
	}
}

// Read the response body and unmarshal it into a interface and returns it
func LoadJsonFromResponse(res *http.Response, format any) error {
	body, err := ReadResBody(res)
	if err != nil {
		return err
	}

	// write to file if debug mode is on
	if DEBUG_MODE {
		logJsonResponse(body)
	}

	if err = json.Unmarshal(body, &format); err != nil {
		err = fmt.Errorf(
			"error %d: failed to unmarshal json response from %s due to %v\nBody: %s",
			RESPONSE_ERROR,
			res.Request.URL.String(),
			err,
			string(body),
		)
		return err
	}
	return nil
}
