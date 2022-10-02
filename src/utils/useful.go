package utils

import (
	"io"
	"fmt"
	"strings"
	"net/http"
	"path/filepath"
	"encoding/json"
)

func SplitArgs(args string) []string {
	if args == "" {
		return []string{}
	}

	splittedArgs := strings.Split(args, ",")
	seen := make(map[string]bool)
	arr := []string{}
	for _, el := range splittedArgs {
		if _, value := seen[el]; !value {
			seen[el] = true
			arr = append(arr, el)
		}
	}
	return arr
}

func GetLastPartOfURL(url string) string {
	removedParams := strings.SplitN(url, "?", 2)
	splittedUrl := strings.Split(removedParams[0], "/")
	return splittedUrl[len(splittedUrl)-1]
}

func ReadResBody(res *http.Response) []byte {
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func JsonPanic(res *http.Response, err error) {
	errorMsg := fmt.Sprintf(
		"failed to parse json response from %s due to %v", 
		res.Request.URL.String(), 
		err,
	)
	panic(errorMsg)
}

func LoadJsonFromResponse(res *http.Response) interface{} {
	body := ReadResBody(res)
	var post interface{}
	if err := json.Unmarshal(body, &post); err != nil {
		JsonPanic(res, err)
	}
	return post
}

func DetectPasswordInText(postFolderPath string, text string) bool {
	passwordFilename := "detected_passwords.txt"
	passwordFilepath := filepath.Join(postFolderPath, passwordFilename)
	for _, passwordText := range PASSWORD_TEXTS {
		if strings.Contains(text, passwordText) {
			passwordText := fmt.Sprintf(
				"Detected a possible password-protected content in post: %s\n\n",
				text,
			)
			LogMessageToPath(passwordText, passwordFilepath)
			return true
		}
	}
	return false
}

func DetectGDriveLinks(text string, isUrl bool, postFolderPath string) bool {
	gdriveFilename := "detected_gdrive_links.txt"
	gdriveFilepath := filepath.Join(postFolderPath, gdriveFilename)
	driveSubstr := "https://drive.google.com"
	containsGDriveLink := false
	if isUrl && strings.HasPrefix(text, driveSubstr) {
		containsGDriveLink = true
	} else if strings.Contains(text, driveSubstr) {
		containsGDriveLink = true
	}

	if !containsGDriveLink {
		return false
	}

	gdriveText := fmt.Sprintf(
		"Google Drive link detected: %s\n\n",
		text,
	)
	LogMessageToPath(gdriveText, gdriveFilepath)
	return true
}

func DetectOtherExtDLLink(text string, postFolderPath string) bool {
	otherExtFilename := "detected_external_links.txt"
	otherExtFilepath := filepath.Join(postFolderPath, otherExtFilename)
	for _, extDownloadProvider := range EXTERNAL_DOWNLOAD_PLATFORMS {
		if strings.Contains(text, extDownloadProvider) {
			otherExtText := fmt.Sprintf(
				"Detected a link that points to an external file hosting in post's description:\n%s\n\n",
				text,
			)
			LogMessageToPath(otherExtText, otherExtFilepath)
			return true
		}
	}
	return false
}