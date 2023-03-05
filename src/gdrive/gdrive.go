package gdrive

import (
	"io"
	"os"
	"fmt"
	"sync"
	"regexp"
	"strings"
	"net/http"
	"crypto/md5"
	"path/filepath"
	"encoding/json"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

const (
	GDRIVE_ERROR_FILENAME = "gdrive_download.log"

	// file fields to fetch from GDrive API:
	// https://developers.google.com/drive/api/v3/reference/files
	GDRIVE_FILE_FIELDS = "id,name,size,mimeType,md5Checksum" 
)

type GDrive struct {
	apiKey string // Google Drive API key to use
	apiUrl string // https://www.googleapis.com/drive/v3/files
	timeout int // timeout in seconds for GDrive API v3
	downloadTimeout int // timeout in seconds for GDrive file downloads
	maxDownloadWorkers int // max concurrent workers for downloading files
}

// Returns a GDrive structure with the given API key and max download workers
func GetNewGDrive(apiKey string, maxDownloadWorkers int) *GDrive {
	gdrive := &GDrive{
		apiKey: apiKey,
		apiUrl: "https://www.googleapis.com/drive/v3/files",
		timeout: 15,
		downloadTimeout: 900, // 15 minutes
		maxDownloadWorkers: maxDownloadWorkers,
	}

	gdriveIsValid, err := gdrive.GDriveKeyIsValid()
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
func (gdrive GDrive) GDriveKeyIsValid() (bool, error) {
	match, _ := regexp.MatchString(`^AIza[\w-]{35}$`, gdrive.apiKey)
	if !match {
		return false, nil
	}

	params := map[string]string{"key": gdrive.apiKey}
	res, err := request.CallRequest("GET", gdrive.apiUrl, gdrive.timeout, nil, nil, params, false)
	if (err != nil) {
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

type GDriveFile struct {
	Kind string `json:"kind"`
	Id string `json:"id"`
	Name string `json:"name"`
	Size string `json:"size"`
	MimeType string `json:"mimeType"`
	Md5Checksum string `json:"md5Checksum"`
}

type GDriveFolder struct {
	Kind string `json:"kind"`
	IncompleteSearch bool `json:"incompleteSearch"`
	Files []GDriveFile `json:"files"`
	NextPageToken string `json:"nextPageToken"`
}

// Logs any failed GDrive API calls to the given log path
func LogFailedGdriveAPICalls(res *http.Response, downloadPath string) {
	requestUrl := res.Request.URL.String()
	errorMsg := fmt.Sprintf(
		"Error while fetching from GDrive...\n" +
		"GDrive URL (May not be accurate): https://drive.google.com/file/d/%s/view?usp=sharing\n" +
		"Status Code: %s\nURL: %s",
		utils.GetLastPartOfURL(requestUrl),
		res.Status,
		requestUrl,
	)
	if downloadPath != "" {
		utils.LogError(nil, errorMsg, false)
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
		utils.LogError(err, "", false)
		return
	}
	defer file.Close()

	// write to file
	_, err = file.WriteString(errorMsg)
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to write to log file, more info => %v",
			utils.OS_ERROR,
			err,
		)
		utils.LogError(err, "", false)
	}
}

// Returns the contents of the given GDrive folder
func (gdrive GDrive) GetFolderContents(folderId, logPath string) ([]map[string]string, error) {
	params := map[string]string{
		"key": gdrive.apiKey,
		"q": fmt.Sprintf("'%s' in parents", folderId),
		"fields": fmt.Sprintf("nextPageToken,files(%s)", GDRIVE_FILE_FIELDS),
	}
	files := []map[string]string{}
	pageToken := ""
	for {
		if pageToken != "" {
			params["pageToken"] = pageToken
		} else {
			delete(params, "pageToken")
		}
		res, err := request.CallRequest("GET", gdrive.apiUrl, gdrive.timeout, nil, nil, params, false)
		if (err != nil) {
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

		gdriveRes := utils.ReadResBody(res)
		gdriveFolder := GDriveFolder{}
		if err := json.Unmarshal(gdriveRes, &gdriveFolder); err != nil {
			err = fmt.Errorf(
				"gdrive error %d: failed to unmarshal GDrive folder contents with ID of %s, more info => %v\nResponse body: %s",
				utils.JSON_ERROR,
				folderId,
				err,
				string(gdriveRes),
			)
			return nil, err
		}
		for _, file := range gdriveFolder.Files {
			files = append(files, map[string]string{
				"id": file.Id,
				"name": file.Name,
				"size": file.Size,
				"mimeType": file.MimeType,
				"md5Checksum": file.Md5Checksum,
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
func (gdrive GDrive) GetNestedFolderContents(folderId, logPath string) ([]map[string]string, error) {
	files := []map[string]string{}
	folderContents, err := gdrive.GetFolderContents(folderId, logPath)
	if err != nil {
		return nil, err
	}

	for _, file := range folderContents {
		if file["mimeType"] == "application/vnd.google-apps.folder" {
			subFolderFiles, err := gdrive.GetNestedFolderContents(file["id"], logPath)
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
func (gdrive GDrive) GetFileDetails(fileId, logPath string) (map[string]string, error) {
	params := map[string]string{
		"key": gdrive.apiKey,
		"fields": GDRIVE_FILE_FIELDS,
	}
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, fileId)
	res, err := request.CallRequest("GET", url, gdrive.timeout, nil, nil, params, false)
	if (err != nil) {
		err = fmt.Errorf(
			"gdrive error %d: failed to get file details with ID of %s, more info => %v",
			utils.CONNECTION_ERROR,
			fileId,
			err,
		)
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res, logPath)
		return nil, nil
	}

	resBody := utils.ReadResBody(res)
	gdriveFile := GDriveFile{}
	if err := json.Unmarshal(resBody, &gdriveFile); err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to unmarshal GDrive file details with ID of %s, more info => %s\nResponse body: %v",
			utils.JSON_ERROR,
			fileId,
			err,
			string(resBody),
		)
		return nil, err
	}
	return map[string]string{
		"id": gdriveFile.Id,
		"name": gdriveFile.Name,
		"size": gdriveFile.Size,
		"mimeType": gdriveFile.MimeType,
		"md5Checksum": gdriveFile.Md5Checksum,
	}, nil
}

// Downloads the given GDrive file using GDrive API v3
//
// If the md5Checksum has a mismatch, the file will be overwritten and downloaded again
func (gdrive GDrive) DownloadFile(fileInfo map[string]string, filePath string) error {
	if utils.PathExists(filePath) {
		// check the md5 checksum and the file size
		file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
		if err != nil {
			err = fmt.Errorf(
				"gdrive error %d: failed to open file \"%s\", more info => %v",
				utils.OS_ERROR,
				filePath,
				err,
			)
			return err
		}
		fileStatInfo, err := file.Stat()
		if err != nil {
			err = fmt.Errorf(
				"gdrive error %d: failed to get file stat info of \"%s\", more info => %v",
				utils.OS_ERROR,
				filePath,
				err,
			)
			return err
		}
		fileSize := fileStatInfo.Size()
		if fmt.Sprintf("%d", fileSize) == fileInfo["size"] {
			md5Checksum := md5.New()
			_, err = io.Copy(md5Checksum, file)
			file.Close()
			if err == nil {
				if fmt.Sprintf("%x", md5Checksum.Sum(nil)) == fileInfo["md5Checksum"] {
					return nil
				}
			}
		} else {
			file.Close()
		}
	}

	params := map[string]string{
		"key": gdrive.apiKey,
		"alt": "media", 			// to tell Google that we are downloading the file
		"acknowledgeAbuse": "true", // If the files are marked as abusive, download them anyway
	}
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, fileInfo["id"])
	res, err := request.CallRequest("GET", url, gdrive.downloadTimeout, nil, nil, params, false)
	if (err != nil) {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res, filePath)
		return nil
	}

	file, err := os.Create(filePath)
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to create file \"%s\", more info => %v",
			utils.OS_ERROR,
			filePath,
			err,
		)
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		os.Remove(filePath)
		err = fmt.Errorf(
			"gdrive error %d: failed to download file \"%s\", more info => %v",
			utils.DOWNLOAD_ERROR,
			filePath,
			err,
		)
		return err
	}
	return nil
}

// Downloads the multiple GDrive file in parallel using GDrive API v3
func (gdrive GDrive) DownloadMultipleFiles(files []map[string]string) {
	var allowedForDownload, notAllowedForDownload []map[string]string
	for _, file := range files {
		if strings.Contains(file["mimeType"], "application/vnd.google-apps") {
			notAllowedForDownload = append(notAllowedForDownload, file)
		} else {
			allowedForDownload = append(allowedForDownload, file)
		}
	}

	if len(notAllowedForDownload) > 0 {
		noticeMsg := "The following files are not allowed for download:\n"
		for _, file := range notAllowedForDownload {
			noticeMsg += fmt.Sprintf(
				"Filename: %s (ID: %s, MIME Type: %s)\n", 
				file["name"], file["id"], file["mimeType"],
			)
		}
		utils.LogError(nil, noticeMsg, false)
	}

	if len(allowedForDownload) > 0 {
		maxConcurrency := gdrive.maxDownloadWorkers
		if len(allowedForDownload) < maxConcurrency {
			maxConcurrency = len(allowedForDownload)
		}

		bar := utils.GetProgressBar(
			len(allowedForDownload),
			"Downloading GDrive files...",
			utils.GetCompletionFunc(
				fmt.Sprintf("Downloaded %d GDrive files", len(allowedForDownload)),
			),
		)
		var wg sync.WaitGroup
		queue := make(chan struct{}, maxConcurrency)
		for _, file := range allowedForDownload {
			wg.Add(1)
			queue <- struct{}{}
			go func(file map[string]string) {
				defer wg.Done()
				os.MkdirAll(file["filepath"], 0755)
				filePath := filepath.Join(file["filepath"], file["name"])
				if err := gdrive.DownloadFile(file, filePath); err != nil {
					errorMsg := fmt.Sprintf(
						"Failed to download file: %s (ID: %s, MIME Type: %s)\nRefer to error details below:\n%v",
						file["name"], file["id"], file["mimeType"], err,
					)
					utils.LogMessageToPath(errorMsg, filepath.Join(file["filepath"], GDRIVE_ERROR_FILENAME))
				}
				bar.Add(1)
				<-queue
			}(file)
		}
		close(queue)
		wg.Wait()
	}
}

// Uses regex to extract the file ID and the file type (type: file, folder) from the given URL
func GetFileIdAndTypeFromUrl(url string) (string, string) {
	matched := utils.GDRIVE_URL_REGEX.FindStringSubmatch(url)
	if matched == nil {
		return "", ""
	}

	var fileType string
	matchedFileType := matched[utils.GDRIVE_REGEX_TYPE_INDEX]
	if strings.Contains(matchedFileType, "folder") {
		fileType = "folder"
	} else if strings.Contains(matchedFileType, "file") {
		fileType = "file"
	} else {
		err := fmt.Errorf(
			"gdrive error %d: could not determine file type from URL, \"%s\"", 
			utils.DEV_ERROR,
			url,
		)
		utils.LogError(err, "", false)
		return "", ""
	}
	return matched[utils.GDRIVE_REGEX_ID_INDEX], fileType
}

// Downloads multiple GDrive files based on a slice of GDrive URL strings in parallel
func (gdrive GDrive) DownloadGdriveUrls(gdriveUrls []map[string]string) error {
	if len(gdriveUrls) == 0 {
		return nil
	}

	// Retrieve the id from the url text
	gdriveIds := []map[string]string{}
	for _, gdriveUrl := range gdriveUrls {
		fileId, fileType := GetFileIdAndTypeFromUrl(gdriveUrl["url"])
		if fileId != "" && fileType != "" {
			gdriveIds = append(gdriveIds, map[string]string{
				"id": fileId,
				"type": fileType,
				"filepath": gdriveUrl["filepath"],
			})
		}
	}

	bar := utils.GetProgressBar(
		len(gdriveIds), 
		"Getting gdrive file info from gdrive ID(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting gdrive file info from %d gdrive ID(s)!", len(gdriveIds))),
	)
	gdriveFilesInfo := []map[string]string{}
	for _, gdriveId := range gdriveIds {
		if gdriveId["type"] == "file" {
			fileInfo, err := gdrive.GetFileDetails(gdriveId["id"], gdriveId["filepath"])
			if err != nil {
				utils.LogMessageToPath(err.Error(), gdriveId["filepath"])
			} else if fileInfo != nil {
				fileInfo["filepath"] = gdriveId["filepath"]
				gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
			}
		} else if gdriveId["type"] == "folder" {
			filesInfo, err := gdrive.GetNestedFolderContents(gdriveId["id"], gdriveId["filepath"])
			if err != nil {
				utils.LogMessageToPath(err.Error(), gdriveId["filepath"])
			} else {
				for _, fileInfo := range filesInfo {
					fileInfo["filepath"] = gdriveId["filepath"]
					gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
				}
			}
		} else {
			errMsg := fmt.Sprintf(
				"gdrive error %d: unknown Google Drive URL type, \"%s\"",
				utils.DEV_ERROR,
				gdriveId["type"],
			)
			utils.LogMessageToPath(errMsg, gdriveId["filepath"])
		}
		bar.Add(1)
	}
	gdrive.DownloadMultipleFiles(gdriveFilesInfo)
	return nil
}