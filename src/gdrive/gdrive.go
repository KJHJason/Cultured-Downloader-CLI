package gdrive

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

const (
	HTTP3_SUPPORTED       = true
	GDRIVE_ERROR_FILENAME = "gdrive_download.log"

	// file fields to fetch from GDrive API:
	// https://developers.google.com/drive/api/v3/reference/files
	GDRIVE_FILE_FIELDS = "id,name,size,mimeType,md5Checksum"
)

type GDriveToDl struct {
	Id 	     string
	Type     string
	FilePath string
}

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
	match, _ := regexp.MatchString(`^AIza[\w-]{35}$`, gdrive.apiKey)
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

// Logs any failed GDrive API calls to the given log path
func LogFailedGdriveAPICalls(res *http.Response, downloadPath string, level int) {
	requestUrl := res.Request.URL.String()
	errorMsg := fmt.Sprintf(
		"Error while fetching from GDrive...\n"+
			"GDrive URL (May not be accurate): https://drive.google.com/file/d/%s/view?usp=sharing\n"+
			"Status Code: %s\nURL: %s",
		utils.GetLastPartOfUrl(requestUrl),
		res.Status,
		requestUrl,
	)
	if downloadPath != "" {
		utils.LogError(nil, errorMsg, false, level)
		return
	}

	// create new text file
	var logFilePath string
	logFilename := GDRIVE_ERROR_FILENAME
	if filepath.Ext(downloadPath) == "" {
		logFilePath = filepath.Join(filepath.Dir(downloadPath), logFilename)
	} else {
		logFilePath = filepath.Join(downloadPath, logFilename)
	}
	file, err := os.OpenFile(
		logFilePath,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0666,
	)
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to open log file, more info => %v",
			utils.OS_ERROR,
			err,
		)
		utils.LogError(err, "", false, utils.ERROR)
		return
	}
	defer file.Close()

	pathLogger := utils.NewLogger(file)
	pathLogger.LogBasedOnLvl(level, errorMsg)
}

type GDriveFile struct {
	Kind        string `json:"kind"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	Size        string `json:"size"`
	MimeType    string `json:"mimeType"`
	Md5Checksum string `json:"md5Checksum"`
}

type GDriveFolder struct {
	Kind             string       `json:"kind"`
	IncompleteSearch bool         `json:"incompleteSearch"`
	Files            []GDriveFile `json:"files"`
	NextPageToken    string       `json:"nextPageToken"`
}

// Returns the contents of the given GDrive folder
func (gdrive *GDrive) GetFolderContents(folderId, logPath string, config *configs.Config) ([]*gdriveFileToDl, error) {
	params := map[string]string{
		"key":    gdrive.apiKey,
		"q":      fmt.Sprintf("'%s' in parents", folderId),
		"fields": fmt.Sprintf("nextPageToken,files(%s)", GDRIVE_FILE_FIELDS),
	}
	var files []*gdriveFileToDl
	pageToken := ""
	for {
		if pageToken != "" {
			params["pageToken"] = pageToken
		} else {
			delete(params, "pageToken")
		}
		res, err := request.CallRequest(
			&request.RequestArgs{
				Url:       gdrive.apiUrl,
				Method:    "GET",
				Timeout:   gdrive.timeout,
				Params:    params,
				UserAgent: config.UserAgent,
				Http2:     !HTTP3_SUPPORTED,
				Http3:     HTTP3_SUPPORTED,
			},
		)
		if err != nil {
			err = fmt.Errorf(
				"gdrive error %d: failed to get folder contents with ID of %s, more info => %v",
				utils.CONNECTION_ERROR,
				folderId,
				err,
			)
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			err = fmt.Errorf(
				"gdrive error %d: failed to get folder contents with ID of %s, more info => %s",
				utils.RESPONSE_ERROR,
				folderId,
				res.Status,
			)
			return nil, err
		}

		var gdriveFolder GDriveFolder
		err = utils.LoadJsonFromResponse(res, &gdriveFolder)
		if err != nil {
			return nil, err
		}

		for _, file := range gdriveFolder.Files {
			files = append(files, &gdriveFileToDl{
				Id:          file.Id,
				Name:        file.Name,
				Size:        file.Size,
				MimeType:    file.MimeType,
				Md5Checksum: file.Md5Checksum,
				FilePath:    "",
			})
		}

		if gdriveFolder.NextPageToken == "" {
			break
		} else {
			pageToken = gdriveFolder.NextPageToken
		}
	}
	return files, nil
}

// Retrieves the content of a GDrive folder and its subfolders recursively using GDrive API v3
func (gdrive *GDrive) GetNestedFolderContents(folderId, logPath string, config *configs.Config) ([]*gdriveFileToDl, error) {
	var files []*gdriveFileToDl
	folderContents, err := gdrive.GetFolderContents(folderId, logPath, config)
	if err != nil {
		return nil, err
	}

	for _, file := range folderContents {
		if file.MimeType == "application/vnd.google-apps.folder" {
			subFolderFiles, err := gdrive.GetNestedFolderContents(file.Id, logPath, config)
			if err != nil {
				return nil, err
			}
			files = append(files, subFolderFiles...)
		} else {
			files = append(files, file)
		}
	}
	return files, nil
}

// Retrieves the file details of the given GDrive file using GDrive API v3
func (gdrive *GDrive) GetFileDetails(gdriveInfo *GDriveToDl, config *configs.Config) (*gdriveFileToDl, error) {
	params := map[string]string{
		"key":    gdrive.apiKey,
		"fields": GDRIVE_FILE_FIELDS,
	}
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, gdriveInfo.Id)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:       url,
			Method:    "GET",
			Timeout:   gdrive.timeout,
			Params:    params,
			UserAgent: config.UserAgent,
			Http2:     !HTTP3_SUPPORTED,
			Http3:     HTTP3_SUPPORTED,
		},
	)
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to get file details with ID of %s, more info => %v",
			utils.CONNECTION_ERROR,
			gdriveInfo.Id,
			err,
		)
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res, gdriveInfo.FilePath, utils.ERROR)
		return nil, nil
	}

	var gdriveFile GDriveFile
	err = utils.LoadJsonFromResponse(res, &gdriveFile)
	if err != nil {
		return nil, err
	}

	return &gdriveFileToDl{
		Id:          gdriveFile.Id,
		Name:        gdriveFile.Name,
		Size:        gdriveFile.Size,
		MimeType:    gdriveFile.MimeType,
		Md5Checksum: gdriveFile.Md5Checksum,
		FilePath:    gdriveInfo.FilePath,
	}, nil
}
