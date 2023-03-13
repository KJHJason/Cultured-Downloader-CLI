package gdrive

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Downloads the given GDrive file using GDrive API v3
//
// If the md5Checksum has a mismatch, the file will be overwritten and downloaded again
func (gdrive *GDrive) DownloadFile(fileInfo map[string]string, filePath, userAgent string, queue chan struct{}) error {
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

	// Create a context that can be cancelled when SIGINT/SIGTERM signal is received
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Catch SIGINT/SIGTERM signal and cancel the context when received
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigs
        cancel()
    }()
	defer signal.Stop(sigs)

	queue <- struct{}{}
	params := map[string]string{
		"key":              gdrive.apiKey,
		"alt":              "media", // to tell Google that we are downloading the file
		"acknowledgeAbuse": "true",  // If the files are marked as abusive, download them anyway
	}
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, fileInfo["id"])
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:	   url,
			Method:	   "GET",
			Timeout:   gdrive.downloadTimeout,
			Params:	   params,
			Context:   ctx,
			UserAgent: userAgent,
		},
	)
	if err != nil {
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
		if fileErr := os.Remove(filePath); fileErr != nil {
			utils.LogError(
				fmt.Errorf(
					"download error %d: failed to remove file at %s, more info => %v",
					utils.OS_ERROR,
					filePath,
					fileErr,
				), 
				"", 
				false,
			)
		}

		if err == context.Canceled {
			return err
		}
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
func (gdrive *GDrive) DownloadMultipleFiles(files []map[string]string, userAgent string) {
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
		errChan := make(chan map[string]string, len(allowedForDownload))

		baseMsg := "Downloading GDrive files [%d/" + fmt.Sprintf("%d]...", len(allowedForDownload))
		progress := spinner.New(
			spinner.DL_SPINNER,
			"fgHiYellow",
			fmt.Sprintf(
				baseMsg,
				0,
			),
			fmt.Sprintf(
				"Finished downloading %d GDrive files!",
				len(allowedForDownload),
			),
			fmt.Sprintf(
				"Something went wrong while downloading %d GDrive files!\nPlease refer to the generated log files for more details.",
				len(allowedForDownload),
			),
			len(allowedForDownload),
		)
		progress.Start()
		for _, file := range allowedForDownload {
			wg.Add(1)
			go func(file map[string]string) {
				defer func() { 
					<-queue
					wg.Done()
				}()

				os.MkdirAll(file["filepath"], 0666)
				filePath := filepath.Join(file["filepath"], file["name"])

				if err := gdrive.DownloadFile(file, filePath, userAgent, queue); err != nil {
					var errorMsg string
					if err == context.Canceled {
						errorMsg = err.Error()
					} else {
						errorMsg = fmt.Sprintf(
							"Failed to download file: %s (ID: %s, MIME Type: %s)\nRefer to error details below:\n%v",
							file["name"], file["id"], file["mimeType"], err,
						)
					}
					errChan <- map[string]string{
						"err":	    errorMsg,
						"filepath": filepath.Join(
							file["filepath"], 
							GDRIVE_ERROR_FILENAME,
						),
					}
				}

				progress.MsgIncrement(baseMsg)
			}(file)
		}
		close(queue)
		wg.Wait()
		close(errChan)

		hasErr := false
		if len(errChan) > 0 {
			hasErr = true
			killProgram := false
			for errMap := range errChan {
				if errMap["err"] == context.Canceled.Error() {
					if !killProgram {
						killProgram = true
					}
					continue
				}

				utils.LogMessageToPath(
					errMap["err"],
					errMap["filepath"], 
				)
			}

			if killProgram {
				progress.KillProgram(
					"Stopped downloading GDrive files (incomplete downloads will be deleted)...",
				)
			}
		}
		progress.Stop(hasErr)
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
func (gdrive *GDrive) DownloadGdriveUrls(gdriveUrls []map[string]string, userAgent string) error {
	if len(gdriveUrls) == 0 {
		return nil
	}

	// Retrieve the id from the url text
	gdriveIds := []map[string]string{}
	for _, gdriveUrl := range gdriveUrls {
		fileId, fileType := GetFileIdAndTypeFromUrl(gdriveUrl["url"])
		if fileId != "" && fileType != "" {
			gdriveIds = append(gdriveIds, map[string]string{
				"id":       fileId,
				"type":     fileType,
				"filepath": gdriveUrl["filepath"],
			})
		}
	}

	var errMapSlice []map[string]string
	gdriveFilesInfo := []map[string]string{}
	baseMsg := "Getting gdrive file info from gdrive ID(s) [%d/" + fmt.Sprintf("%d]...", len(gdriveIds))
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting gdrive file info from %d gdrive ID(s)!",
			len(gdriveIds),
		),
		fmt.Sprintf(
			"Something went wrong while getting gdrive file info from %d gdrive ID(s)!\nPlease refer to the generated log files for more details.",
			len(gdriveIds),
		),
		len(gdriveIds),
	)
	progress.Start()
	for _, gdriveId := range gdriveIds {
		if gdriveId["type"] == "file" {
			fileInfo, err := gdrive.GetFileDetails(
				gdriveId["id"],
				gdriveId["filepath"],
				userAgent,
			)
			if err != nil {
				errMapSlice = append(errMapSlice, map[string]string{
					"err":	    err.Error(),
					"filepath": gdriveId["filepath"],
				})
			} else if fileInfo != nil {
				fileInfo["filepath"] = gdriveId["filepath"]
				gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
			}
			progress.MsgIncrement(baseMsg)
			continue
		} 

		if gdriveId["type"] == "folder" {
			filesInfo, err := gdrive.GetNestedFolderContents(
				gdriveId["id"], 
				gdriveId["filepath"],
				userAgent,
			)
			if err != nil {
				errMapSlice = append(errMapSlice, map[string]string{
					"err":	    err.Error(),
					"filepath": gdriveId["filepath"],
				})
			} else {
				for _, fileInfo := range filesInfo {
					fileInfo["filepath"] = gdriveId["filepath"]
					gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
				}
			}
			progress.MsgIncrement(baseMsg)
			continue
		} 

		errMsg := fmt.Sprintf(
			"gdrive error %d: unknown Google Drive URL type, \"%s\"",
			utils.DEV_ERROR,
			gdriveId["type"],
		)
		errMapSlice = append(errMapSlice, map[string]string{
			"err":	    errMsg,
			"filepath": gdriveId["filepath"],
		})
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errMapSlice) > 0 {
		hasErr = true
		for _, errMap := range errMapSlice {
			utils.LogMessageToPath(
				errMap["err"],
				errMap["filepath"], 
			)
		}
	}
	progress.Stop(hasErr)

	gdrive.DownloadMultipleFiles(gdriveFilesInfo, userAgent)
	return nil
}
