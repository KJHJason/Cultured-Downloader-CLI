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

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type gdriveFileToDl struct {
	Id          string
	Name        string
	Size        string
	MimeType    string
	Md5Checksum string
	FilePath    string
}

// Process and detects for any external download links from the post's text content
func ProcessPostText(postBodyStr, postFolderPath string, downloadGdrive bool) []*request.ToDownload {
	if postBodyStr == "" {
		return nil
	}

	// split the text by newlines
	postBodySlice := strings.FieldsFunc(
		postBodyStr,
		func(c rune) bool {
			return c == '\n'
		},
	)
	loggedPassword := false
	var detectedGdriveLinks []*request.ToDownload
	for _, text := range postBodySlice {
		if utils.DetectPasswordInText(text) && !loggedPassword {
			// Log the entire post text if it contains a password
			filePath := filepath.Join(postFolderPath, utils.PASSWORD_FILENAME)
			if !utils.PathExists(filePath) {
				loggedPassword = true
				postBodyStr := strings.Join(postBodySlice, "\n")
				utils.LogMessageToPath(
					"Found potential password in the post:\n\n"+postBodyStr,
					filePath,
					utils.ERROR,
				)
			}
		}

		utils.DetectOtherExtDLLink(text, postFolderPath)
		if utils.DetectGDriveLinks(text, postFolderPath, false) && downloadGdrive {
			detectedGdriveLinks = append(detectedGdriveLinks, &request.ToDownload{
				Url:      text,
				FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
			})
		}
	}
	return detectedGdriveLinks
}

// Downloads the given GDrive file using GDrive API v3
//
// If the md5Checksum has a mismatch, the file will be overwritten and downloaded again
func (gdrive *GDrive) DownloadFile(fileInfo *gdriveFileToDl, filePath string, config *configs.Config, queue chan struct{}) error {
	if fileInfo == nil {
		return nil
	}

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
		if fmt.Sprintf("%d", fileSize) == fileInfo.Size {
			md5Checksum := md5.New()
			_, err = io.Copy(md5Checksum, file)
			file.Close()
			if err == nil {
				if fmt.Sprintf("%x", md5Checksum.Sum(nil)) == fileInfo.Md5Checksum {
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
	url := fmt.Sprintf("%s/%s", gdrive.apiUrl, fileInfo.Id)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:       url,
			Method:    "GET",
			Timeout:   gdrive.downloadTimeout,
			Params:    params,
			Context:   ctx,
			UserAgent: config.UserAgent,
		},
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		LogFailedGdriveAPICalls(res, filePath, utils.ERROR)
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
				utils.ERROR,
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

type gdriveError struct {
	err      error
	filePath string
}

// Downloads the multiple GDrive file in parallel using GDrive API v3
func (gdrive *GDrive) DownloadMultipleFiles(files []*gdriveFileToDl, config *configs.Config) {
	var allowedForDownload, notAllowedForDownload []*gdriveFileToDl
	for _, file := range files {
		if strings.Contains(file.MimeType, "application/vnd.google-apps") {
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
				file.Name, file.Id, file.MimeType,
			)
		}
		utils.LogError(nil, noticeMsg, false, utils.INFO)
	}

	if len(allowedForDownload) > 0 {
		maxConcurrency := gdrive.maxDownloadWorkers
		if len(allowedForDownload) < maxConcurrency {
			maxConcurrency = len(allowedForDownload)
		}
		var wg sync.WaitGroup
		queue := make(chan struct{}, maxConcurrency)
		errChan := make(chan *gdriveError, len(allowedForDownload))

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
			go func(file *gdriveFileToDl) {
				defer func() {
					<-queue
					wg.Done()
				}()

				os.MkdirAll(file.FilePath, 0666)
				filePath := filepath.Join(file.FilePath, file.Name)

				if err := gdrive.DownloadFile(file, filePath, config, queue); err != nil {
					if err != context.Canceled {
						err = fmt.Errorf(
							"failed to download file: %s (ID: %s, MIME Type: %s)\nRefer to error details below:\n%v",
							file.Name, file.Id, file.MimeType, err,
						)
					}
					errChan <- &gdriveError{
						err: err,
						filePath: filepath.Join(
							file.FilePath,
							GDRIVE_ERROR_FILENAME,
						),
					}
				}

				progress.MsgIncrement(baseMsg)
			}(file)
		}
		wg.Wait()
		close(queue)
		close(errChan)

		hasErr := false
		if len(errChan) > 0 {
			hasErr = true
			killProgram := false
			for errInfo := range errChan {
				errMsg := errInfo.err.Error()
				if errMsg == context.Canceled.Error() {
					if !killProgram {
						killProgram = true
					}
					continue
				}

				utils.LogMessageToPath(
					errMsg,
					errInfo.filePath,
					utils.ERROR,
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
		utils.LogError(err, "", false, utils.ERROR)
		return "", ""
	}
	return matched[utils.GDRIVE_REGEX_ID_INDEX], fileType
}

// Downloads multiple GDrive files based on a slice of GDrive URL strings in parallel
func (gdrive *GDrive) DownloadGdriveUrls(gdriveUrls []*request.ToDownload, config *configs.Config) error {
	if len(gdriveUrls) == 0 {
		return nil
	}

	// Retrieve the id from the url text
	var gdriveIds []*GDriveToDl
	for _, gdriveUrl := range gdriveUrls {
		fileId, fileType := GetFileIdAndTypeFromUrl(gdriveUrl.Url)
		if fileId != "" && fileType != "" {
			gdriveIds = append(gdriveIds, &GDriveToDl{
				Id:       fileId,
				Type:     fileType,
				FilePath: gdriveUrl.FilePath,
			})
		}
	}

	var errSlice []*gdriveError
	var gdriveFilesInfo []*gdriveFileToDl
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
		if gdriveId.Type == "file" {
			fileInfo, err := gdrive.GetFileDetails(
				gdriveId.Id,
				gdriveId.FilePath,
				config,
			)
			if err != nil {
				errSlice = append(errSlice, &gdriveError{
					err:      err,
					filePath: gdriveId.FilePath,
				})
			} else if fileInfo != nil {
				fileInfo.FilePath = gdriveId.FilePath
				gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
			}
			progress.MsgIncrement(baseMsg)
			continue
		}

		if gdriveId.Type == "folder" {
			filesInfo, err := gdrive.GetNestedFolderContents(
				gdriveId.Id,
				gdriveId.FilePath,
				config,
			)
			if err != nil {
				errSlice = append(errSlice, &gdriveError{
					err:      err,
					filePath: gdriveId.FilePath,
				})
			} else {
				for _, fileInfo := range filesInfo {
					fileInfo.FilePath = gdriveId.FilePath
					gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
				}
			}
			progress.MsgIncrement(baseMsg)
			continue
		}

		errSlice = append(errSlice, &gdriveError{
			err: fmt.Errorf(
				"gdrive error %d: unknown Google Drive URL type, \"%s\"",
				utils.DEV_ERROR,
				gdriveId.Type,
			),
			filePath: gdriveId.FilePath,
		})
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		for _, err := range errSlice {
			utils.LogMessageToPath(
				err.err.Error(),
				err.filePath,
				utils.ERROR,
			)
		}
	}
	progress.Stop(hasErr)

	gdrive.DownloadMultipleFiles(gdriveFilesInfo, config)
	return nil
}
