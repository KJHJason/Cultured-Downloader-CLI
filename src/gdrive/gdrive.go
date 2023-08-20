package gdrive

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	HTTP3_SUPPORTED        = true
	GDRIVE_ERROR_FILENAME  = "gdrive_download.log"
	BASE_API_KEY_REGEX_STR = `AIza[\w-]{35}`

	// file fields to fetch from GDrive API:
	// https://developers.google.com/drive/api/v3/reference/files
	GDRIVE_FILE_FIELDS = "id,name,size,mimeType,md5Checksum"
	GDRIVE_FOLDER_FIELDS = "nextPageToken,files(id,name,size,mimeType,md5Checksum)"
)

var (
	API_KEY_REGEX = regexp.MustCompile(fmt.Sprintf(`^%s$`, BASE_API_KEY_REGEX_STR))
	API_KEY_PARAM_REGEX = regexp.MustCompile(fmt.Sprintf(`key=%s`, BASE_API_KEY_REGEX_STR))
)

type GDrive struct {
	apiKey             string         // Google Drive API key to use
	client             *drive.Service // Google Drive service client (if using service account credentials)
	apiUrl             string         // https://www.googleapis.com/drive/v3/files
	timeout            int            // timeout in seconds for GDrive API v3
	downloadTimeout    int            // timeout in seconds for GDrive file downloads
	maxDownloadWorkers int            // max concurrent workers for downloading files
}

// Returns a GDrive structure with the given API key and max download workers
func GetNewGDrive(apiKey, jsonPath string, config *configs.Config, maxDownloadWorkers int) *GDrive {
	if jsonPath != "" && apiKey != "" {
		color.Red("Both Google Drive API key and service account credentials file cannot be used at the same time.")
		os.Exit(1)
	} else if jsonPath == "" && apiKey == "" {
		color.Red("Google Drive API key or service account credentials file is required.")
		os.Exit(1)
	}

	gdrive := &GDrive{
		apiUrl:             "https://www.googleapis.com/drive/v3/files",
		timeout:            15,
		downloadTimeout:    900, // 15 minutes
		maxDownloadWorkers: maxDownloadWorkers,
	}
	if apiKey != "" {
		gdrive.apiKey = apiKey
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

	if !utils.PathExists(jsonPath) {
		color.Red("Unable to access Drive API due to missing credentials file: %s", jsonPath)
		os.Exit(1)
	}
	srv, err := drive.NewService(context.Background(), option.WithCredentialsFile(jsonPath))
	if err != nil {
		color.Red("Unable to access Drive API due to %v", err)
		os.Exit(1)
	}
	gdrive.client = srv
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
		return false, fmt.Errorf(
			"gdrive error %d: failed to check if Google Drive API key is valid, more info => %v",
			utils.CONNECTION_ERROR,
			err,
		)
	}
	res.Body.Close()
	return res.StatusCode != 400, nil
}
