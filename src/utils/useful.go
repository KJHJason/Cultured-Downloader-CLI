package utils

import (
	"io"
	"os"
	"fmt"
	"errors"
	"strings"
	"net/http"
	"archive/zip"
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

// based on https://stackoverflow.com/a/24792688/2737403
func UnzipFile(src, dest string, ignoreIfMissing bool) error {
	if !PathExists(src) {
		if ignoreIfMissing {
			return nil
		} else {
			return errors.New("source zip file does not exist")
		}
	}

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)
	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)
		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest) + string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}
	return nil
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

func DetectPasswordInText(postFolderPath, text string) bool {
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

func DetectGDriveLinks(text, postFolderPath string, isUrl bool) bool {
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

func DetectOtherExtDLLink(text, postFolderPath string) bool {
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