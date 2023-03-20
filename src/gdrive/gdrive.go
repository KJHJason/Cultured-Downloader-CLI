package gdrive

import (
	"fmt"
	"os"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

const (
	HTTP3_SUPPORTED        = true
	GDRIVE_ERROR_FILENAME  = "gdrive_download.log"
	BASE_API_KEY_REGEX_STR = `AIza[\w-]{35}`

	// file fields to fetch from GDrive API:
	// https://developers.google.com/drive/api/v3/reference/files
	GDRIVE_FILE_FIELDS = "id,name,size,mimeType,md5Checksum"
)

var (
	API_KEY_REGEX = regexp.MustCompile(fmt.Sprintf(`^%s$`, BASE_API_KEY_REGEX_STR))
	API_KEY_PARAM_REGEX = regexp.MustCompile(fmt.Sprintf(`key=%s`, BASE_API_KEY_REGEX_STR))
)

type GDrive struct {
	apiKey             string // Google Drive API key to use
	apiUrl             string // https://www.googleapis.com/drive/v3/files
	timeout            int    // timeout in seconds for GDrive API v3
	downloadTimeout    int    // timeout in seconds for GDrive file downloads
	maxDownloadWorkers int    // max concurrent workers for downloading files
}

// Returns a GDrive structure with the given API key and max download workers
func GetNewGDrive(apiKey string, config *configs.Config, maxDownloadWorkers int) *GDrive {
	gdrive := &GDrive{
		apiKey:             apiKey,
		apiUrl:             "https://www.googleapis.com/drive/v3/files",
		timeout:            15,
		downloadTimeout:    900, // 15 minutes
		maxDownloadWorkers: maxDownloadWorkers,
	}

	gdriveIsValid, err := gdrive.GDriveKeyIsValid(config.UserAgent)
	if err != nil {
		color.Red(err.Error())
		os.Exit(1)
	} else if !gdriveIsValid {
		color.Red("Google Drive API key is invalid.")
		os.Exit(1)
	}
	return gdrive
}

// Checks if the given Google Drive API key is valid
//
// Will return true if the given Google Drive API key is valid
func (gdrive *GDrive) GDriveKeyIsValid(userAgent string) (bool, error) {
	match := API_KEY_REGEX.MatchString(gdrive.apiKey)
	if !match {
		return false, nil
	}

	params := map[string]string{"key": gdrive.apiKey}
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:       gdrive.apiUrl,
			Method:    "GET",
			Timeout:   gdrive.timeout,
			Params:    params,
			UserAgent: userAgent,
			Http2:     !HTTP3_SUPPORTED,
			Http3:     HTTP3_SUPPORTED,
		},
	)
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to check if Google Drive API key is valid, more info => %v",
			utils.CONNECTION_ERROR,
			err,
		)
		return false, err
	}
	res.Body.Close()
	return res.StatusCode != 400, nil
}
