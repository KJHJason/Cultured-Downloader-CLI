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

type GDrive struct {
	apiKey string // Google Drive API key to use
	apiUrl string // https://www.googleapis.com/drive/v3/files
	timeout int // timeout in seconds for GDrive API v3
	downloadTimeout int // timeout in seconds for GDrive file downloads
	maxDownloadWorkers int // max concurrent workers for downloading files
}

func GetNewGDrive(apiKey string, maxDownloadWorkers int) *GDrive {
	gdrive := &GDrive{
		apiKey: apiKey,
		apiUrl: "https://www.googleapis.com/drive/v3/files",
		timeout: 15,
		downloadTimeout: 900, // 15 minutes
		maxDownloadWorkers: maxDownloadWorkers,
	}
	if !gdrive.GDriveKeyIsValid() {
		color.Red("Google Drive API key is invalid.")
		os.Exit(1)
	}
	return gdrive
}

func (gdrive GDrive) GDriveKeyIsValid() bool {
	match, _ := regexp.MatchString(`^AIza[\w-]{35}$`, gdrive.apiKey)
	if !match {
		return false
	}

	params := map[string]string{"key": gdrive.apiKey}
	res, err := request.CallRequest("GET", gdrive.apiUrl, gdrive.timeout, nil, nil, params, false)
	if (err != nil) {
		panic(err)
	}
	res.Body.Close()
	return res.StatusCode != 400
}

type GDriveFile struct {
	Kind string `json:"kind"`
	Id string `json:"id"`
	Name string `json:"name"`
	MimeType string `json:"mimeType"`
	Md5Checksum string `json:"md5Checksum"`
}

type GDriveFolder struct {
	Kind string `json:"kind"`
	IncompleteSearch bool `json:"incompleteSearch"`
	Files []GDriveFile `json:"files"`
	NextPageToken string `json:"nextPageToken"`
}

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
	logFilename := "gdrive_download.log"
	if filepath.Ext(downloadPath) == "" {
		logFilePath = filepath.Join(filepath.Dir(downloadPath), logFilename)
	} else {
		logFilePath = filepath.Join(downloadPath, logFilename)
	}
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// write to file
	_, err = file.WriteString(errorMsg)
	if err != nil {
		panic(err)
	}
}

func (gdrive GDrive) GetFolderContents(folderId, logPath string) []map[string]string {
	params := map[string]string{
		"key": gdrive.apiKey,
		"q": fmt.Sprintf("'%s' in parents", folderId),
		"fields": "nextPageToken,files(id,name,mimeType,md5Checksum)",
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
			panic(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			LogFailedGdriveAPICalls(res, logPath)
			return nil
		}

		gdriveRes := utils.ReadResBody(res)
		gdriveFolder := GDriveFolder{}
		if err := json.Unmarshal(gdriveRes, &gdriveFolder); err != nil {
			panic(err)
		}
		for _, file := range gdriveFolder.Files {
			files = append(files, map[string]string{
				"id": file.Id,
				"name": file.Name,
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
	return files
}

func (gdrive GDrive) GetNestedFolderContents(folderId, logPath string) []map[string]string {
	files := []map[string]string{}
	folderContents := gdrive.GetFolderContents(folderId, logPath)
	for _, file := range folderContents {
		if file["mimeType"] == "application/vnd.google-apps.folder" {
			files = append(files, gdrive.GetNestedFolderContents(file["id"], logPath)...)
		} else {
			files = append(files, file)
		}
	}
	return files
}

func (gdrive GDrive) GetFileDetails(fileId, logPath string) map[string]string {
	params := map[string]string{
		"key": gdrive.apiKey,
		"fields": "id,name,mimeType,md5Checksum",
	}
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, fileId)
	res, err := request.CallRequest("GET", url, gdrive.timeout, nil, nil, params, false)
	if (err != nil) {
		panic(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res, logPath)
		return nil
	}

	gdriveFile := GDriveFile{}
	if err := json.Unmarshal(utils.ReadResBody(res), &gdriveFile); err != nil {
		panic(err)
	}
	return map[string]string{
		"id": gdriveFile.Id,
		"name": gdriveFile.Name,
		"mimeType": gdriveFile.MimeType,
		"md5Checksum": gdriveFile.Md5Checksum,
	}
}

func (gdrive GDrive) DownloadFile(fileInfo map[string]string, filePath string) error {
	if utils.PathExists(filePath) {
		// check the md5 checksum
		file, err := os.Open(filePath)
		if err == nil {
			md5Hash := md5.New()
			_, err := io.Copy(md5Hash, file)
			file.Close()
			if err == nil {
				if fmt.Sprintf("%x", md5Hash.Sum(nil)) == fileInfo["md5Checksum"] {
					return nil
				}
			}
		}
	}

	params := map[string]string{
		"key": gdrive.apiKey,
		"alt": "media", // to tell Google that we are downloading the file
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
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		os.Remove(filePath)
		panic(err)
	}
	return nil
}

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
						"Failed to download file: %s (ID: %s, MIME Type: %s)\nError Details: %v",
						file["name"], file["id"], file["mimeType"], err,
					)
					utils.LogError(err, errorMsg, true)
				}
				<-queue
			}(file)
		}
		close(queue)
		wg.Wait()
	}
}

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
		panic(fmt.Sprintf("Could not determine file type from URL: %s", url))
	}
	return matched[utils.GDRIVE_REGEX_ID_INDEX], fileType
}

func (gdrive GDrive) DownloadGdriveUrls(gdriveUrls []map[string]string) {
	if len(gdriveUrls) == 0 {
		return
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

	gdriveFilesInfo := []map[string]string{}
	for _, gdriveId := range gdriveIds {
		if gdriveId["type"] == "file" {
			fileInfo := gdrive.GetFileDetails(gdriveId["id"], gdriveId["filepath"])
			fileInfo["filepath"] = gdriveId["filepath"]
			gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
		} else if gdriveId["type"] == "folder" {
			filesInfo := gdrive.GetNestedFolderContents(gdriveId["id"], gdriveId["filepath"])
			for _, fileInfo := range filesInfo {
				fileInfo["filepath"] = gdriveId["filepath"]
				gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
			}
		} else {
			panic(fmt.Sprintf("Unknown Google Drive URL type: %s", gdriveId["type"]))
		}
	}

	gdrive.DownloadMultipleFiles(gdriveFilesInfo)
}