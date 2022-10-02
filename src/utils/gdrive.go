package utils

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
	"github.com/panjf2000/ants/v2"
)

type GDrive struct {
	apiKey string // Google Drive API key to use
	apiUrl string // https://www.googleapis.com/drive/v3/files
	timeout int // timeout in seconds for GDrive API v3
	maxDownloadWorkers int // max concurrent workers for downloading files
}

func GetNewGDrive(apiKey string, maxDownloadWorkers int) *GDrive {
	gdrive := &GDrive{
		apiKey: apiKey,
		apiUrl: "https://www.googleapis.com/drive/v3/files",
		timeout: 15,
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
	res, err := CallRequest(gdrive.apiUrl, gdrive.timeout, nil, "GET", nil, params, false)
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

func LogFailedGdriveAPICalls(res *http.Response) {
	resBody, err := io.ReadAll(res.Body)
	var resBodyStr string
	if (err != nil) {
		resBodyStr = "Unable to read response body"
	} else {
		resBodyStr = string(resBody)
	}
	errorMsg := fmt.Sprintf(
		"Error while fetching from GDrive...\n" +
		"Status Code: %s\n" + "JSON Response: %s",
		res.Status,
		resBodyStr,
	)
	LogError(nil, errorMsg, false)
}

func (gdrive GDrive) GetFolderContents(folderId string) []map[string]string {
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
		res, err := CallRequest(gdrive.apiUrl, gdrive.timeout, nil, "GET", nil, params, false)
		if (err != nil) {
			panic(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			LogFailedGdriveAPICalls(res)
			return []map[string]string{}
		}

		gdriveRes := ReadResBody(res)
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

func (gdrive GDrive) GetNestedFolderContents(folderId string) []map[string]string {
	files := []map[string]string{}
	folderContents := gdrive.GetFolderContents(folderId)
	for _, file := range folderContents {
		if file["mimeType"] == "application/vnd.google-apps.folder" {
			files = append(files, gdrive.GetNestedFolderContents(file["id"])...)
		} else {
			files = append(files, file)
		}
	}
	return files
}

func (gdrive GDrive) GetFileDetails(fileId string) map[string]string {
	params := map[string]string{
		"key": gdrive.apiKey,
		"fields": "id,name,mimeType,md5Checksum",
	}
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, fileId)
	res, err := CallRequest(url, gdrive.timeout, nil, "GET", nil, params, false)
	if (err != nil) {
		panic(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res)
		return map[string]string{}
	}

	gdriveFile := GDriveFile{}
	if err := json.Unmarshal(ReadResBody(res), &gdriveFile); err != nil {
		panic(err)
	}
	return map[string]string{
		"id": gdriveFile.Id,
		"name": gdriveFile.Name,
		"mimeType": gdriveFile.MimeType,
		"md5Checksum": gdriveFile.Md5Checksum,
	}
}

func (gdrive GDrive) DownloadFile(fileInfo map[string]string, folderPath string) error {
	filePath := filepath.Join(folderPath, fileInfo["name"])
	if PathExists(filePath) {
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
	res, err := CallRequest(url, gdrive.timeout, nil, "GET", nil, params, false)
	if (err != nil) {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res)
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
	var allowedForDownload []map[string]string
	var notAllowedForDownload []map[string]string
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
		LogError(nil, noticeMsg, false)
	}

	if len(allowedForDownload) > 0 {
		maxConcurrency := gdrive.maxDownloadWorkers
		if len(allowedForDownload) < maxConcurrency {
			maxConcurrency = len(allowedForDownload)
		}

		var wg sync.WaitGroup
		pool, _ := ants.NewPool(maxConcurrency)
		for _, file := range allowedForDownload {
			wg.Add(1)
			_ = pool.Submit(func() {
				defer wg.Done()
				filePath := filepath.Join(file["filepath"], file["name"])
				if err := gdrive.DownloadFile(file, filePath); err != nil {
					errorMsg := fmt.Sprintf(
						"Failed to download file: %s (ID: %s, MIME Type: %s)",
						file["name"], file["id"], file["mimeType"],
					)
					LogError(err, errorMsg, true)
				}
			})
		}
		wg.Wait()
	}
}

func GetFileIdAndTypeFromUrl(url string) (string, string) {
	matched := GDRIVE_URL_REGEX.FindStringSubmatch(url)
	if matched == nil {
		return "", ""
	}

	var fileType string
	matchedFileType := matched[GDRIVE_REGEX_TYPE_INDEX]
	if strings.Contains(matchedFileType, "folder") {
		fileType = "folder"
	} else if strings.Contains(matchedFileType, "file") {
		fileType = "file"
	} else {
		panic(fmt.Sprintf("Could not determine file type from URL: %s", url))
	}
	return matched[GDRIVE_REGEX_ID_INDEX], fileType
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
			fileInfo := gdrive.GetFileDetails(gdriveId["id"])
			fileInfo["filepath"] = gdriveId["filepath"]
			gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
		} else if gdriveId["type"] == "folder" {
			filesInfo := gdrive.GetNestedFolderContents(gdriveId["id"])
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