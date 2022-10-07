package utils

import (
	"io"
	"os"
	"fmt"
	"time"
	"errors"
	"strings"
	"net/url"
	"net/http"
	"math/rand"
	"archive/zip"
	"path/filepath"
	"encoding/json"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Returns a function that will print out a success message in green with a checkmark
//
// To be used with progressbar.OptionOnCompletion (github.com/schollz/progressbar/v3)
func GetCompletionFunc(completionMsg string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, color.GreenString("\033[2K\r✓ %s\n"), completionMsg)
	}
}

// Returns the ProgressBar structure for printing a progress bar for the user to see
func GetProgressBar(total int, desc string, completionFunc func()) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		int64(total),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(completionFunc),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetRenderBlankState(true),
	)
}

// Returns a random time.Duration between the given min and max arguments
func GetRandomTime(min, max float64) time.Duration {
	rand.Seed(time.Now().UnixNano())
	randomDelay := min + rand.Float64() * (max - min)
	return time.Duration(randomDelay * 1000) * time.Millisecond 
}

// Returns a random time.Duration between the defined min and max delay values in the contants.go file
func GetRandomDelay() time.Duration {
	return GetRandomTime(MIN_RETRY_DELAY, MAX_RETRY_DELAY)
}

// Checks if the given str is in the given arr and returns a boolean
func ArrContains(arr []string, str string) bool {
	for _, el := range arr {
		if el == str {
			return true
		}
	}
	return false
}

// Checks if the slice of string contains the target str
//
// Otherwise, os.Exit(1) is called after printing error messages for the user to read
func CheckStrArgs(arr []string, str, argName string) *string {
	str = strings.ToLower(str)
	if ArrContains(arr, str) {
		return &str
	} else {
		color.Red("Invalid %s: %s", argName, str)
		color.Red(
			fmt.Sprintf(
				"Valid %ss: %s",
				argName,
				strings.TrimSpace(strings.Join(arr, ", ")),
			),
		)
		os.Exit(1)
		return nil
	}
}

// Since go's flags package doesn't have a way to pass in multiple values for a flag,
// a workaround is to pass in the args like "arg1 arg2 arg3" and 
// then split them into a slice by the space delimiter
//
// This function will split based on the given delimiter and return the slice of strings
func SplitArgsWithSep(args, sep string) []string {
	if args == "" {
		return []string{}
	}

	splittedArgs := strings.Split(args, sep)
	seen := make(map[string]bool)
	arr := []string{}
	for _, el := range splittedArgs {
		el = strings.TrimSpace(el)
		if _, value := seen[el]; !value {
			seen[el] = true
			arr = append(arr, el)
		}
	}
	return arr
}

// The same as SplitArgsWithSep, but with the default space delimiter
func SplitArgs(args string) []string {
	return SplitArgsWithSep(args, " ")
}

// Split the given string into a slice of strings with space as the delimiter
// and checks if the slice of strings contains only numbers
//
// Otherwise, os.Exit(1) is called after printing error messages for the user to read
func SplitAndCheckIds(args string) []string {
	ids := SplitArgs(args)
	for _, id := range ids {
		if !NUMBER_REGEX.MatchString(id) {
			color.Red("Invalid ID: %s", id)
			color.Red("IDs must be numbers!")
			os.Exit(1)
		}
	}
	return ids
}

// Same as strings.Join([]string, "\n")
func CombineStringsWithNewline(strs []string) string {
	return strings.Join(strs, "\n")
}

// Unzips the given zip file to the given destination
//
// Code based on https://stackoverflow.com/a/24792688/2737403
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

// Returns the last part of the given URL string
func GetLastPartOfURL(url string) string {
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
func ReadResBody(res *http.Response) []byte {
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return body
}

// Prints out a defined error message and exits the program
func JsonPanic(res *http.Response, err error) {
	errorMsg := fmt.Sprintf(
		"failed to parse json response from %s due to %v", 
		res.Request.URL.String(), 
		err,
	)
	panic(errorMsg)
}

// Read the response body and unmarshal it into a interface and returns it
func LoadJsonFromResponse(res *http.Response) interface{} {
	body := ReadResBody(res)
	var post interface{}
	if err := json.Unmarshal(body, &post); err != nil {
		JsonPanic(res, err)
	}
	return post
}

// Detects if the given string contains any passwords
func DetectPasswordInText(text string) bool {
	for _, passwordText := range PASSWORD_TEXTS {
		if strings.Contains(text, passwordText) {
			return true
		}
	}
	return false
}

// Detects if the given string contains any GDrive links and logs it if detected
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

// Detects if the given string contains any other external file hosting providers links and logs it if detected
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