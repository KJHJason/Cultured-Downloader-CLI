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
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive/models"
)

func md5HashFile(file *os.File) (string, error) {
	md5Checksum := md5.New()
	_, err := io.Copy(md5Checksum, file)
	if err != nil {
		return "", fmt.Errorf(
			"gdrive error %d: failed to calculate file's md5 checksum, more info => %v",
			utils.OS_ERROR,
			err,
		)
	}
	return fmt.Sprintf("%x", md5Checksum.Sum(nil)), nil
}

func checkIfCanSkipDl(filePath string, fileInfo *models.GdriveFileToDl) (bool, error) {
	if !utils.PathExists(filePath) {
		return false, nil
	}

	// check the md5 checksum and the file size
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to open file %q, more info => %v",
			utils.OS_ERROR,
			filePath,
			err,
		)
		return false, err
	}
	defer file.Close()

	fileStatInfo, err := file.Stat()
	if err != nil {
		err = fmt.Errorf(
			"gdrive error %d: failed to get file stat info of %q, more info => %v",
			utils.OS_ERROR,
			filePath,
			err,
		)
		return false, err
	}

	fileSize := fileStatInfo.Size()
	if strconv.FormatInt(fileSize, 10) != fileInfo.Size {
		return false, nil
	}

	md5Checksum, err := md5HashFile(file)
	if err != nil {
		return false, err
	}
	return md5Checksum == fileInfo.Md5Checksum, nil
}

// Downloads the given GDrive file using GDrive API v3
//
// If the md5Checksum has a mismatch, the file will be overwritten and downloaded again
func (gdrive *GDrive) DownloadFile(fileInfo *models.GdriveFileToDl, filePath string, config *configs.Config, queue chan struct{}) error {
	skipDl, err := checkIfCanSkipDl(filePath, fileInfo)
	if skipDl || err != nil {
		return err
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
			Http2:     !HTTP3_SUPPORTED,
			Http3:     HTTP3_SUPPORTED,
		},
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return getFailedApiCallErr(res)
	}
	return request.DlToFile(res, url, filePath)
}

func filterDownloads(files []*models.GdriveFileToDl) []*models.GdriveFileToDl {
	var notAllowedForDownload []*models.GdriveFileToDl
	allowedForDownload := make([]*models.GdriveFileToDl, 0, len(files))
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
	return allowedForDownload
}

func processGdriveDlError(errChan chan *models.GdriveError, progress *spinner.Spinner) {
	killProgram := false
	for errInfo := range errChan {
		errMsg := censorApiKeyFromStr(errInfo.Err.Error())
		if errMsg == context.Canceled.Error() {
			if !killProgram {
				killProgram = true
			}
			continue
		}

		utils.LogMessageToPath(
			censorApiKeyFromStr(errMsg),
			errInfo.FilePath,
			utils.ERROR,
		)
	}

	if killProgram {
		progress.KillProgram(
			"Stopped downloading GDrive files (incomplete downloads will be deleted)...",
		)
	}
}

// Downloads the multiple GDrive file in parallel using GDrive API v3
func (gdrive *GDrive) DownloadMultipleFiles(files []*models.GdriveFileToDl, config *configs.Config) {
	allowedForDownload := filterDownloads(files)
	if len(allowedForDownload) == 0 {
		return
	}

	maxConcurrency := gdrive.maxDownloadWorkers
	if len(allowedForDownload) < maxConcurrency {
		maxConcurrency = len(allowedForDownload)
	}
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	errChan := make(chan *models.GdriveError, len(allowedForDownload))

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
		go func(file *models.GdriveFileToDl) {
			defer func() {
				wg.Done()
				<-queue
			}()

			os.MkdirAll(file.FilePath, 0666)
			filePath := filepath.Join(file.FilePath, file.Name)

			err := gdrive.DownloadFile(file, filePath, config, queue)
			if err != nil && err != context.Canceled {
				err = fmt.Errorf(
					"failed to download file: %s (ID: %s, MIME Type: %s)\nRefer to error details below:\n%v",
					file.Name, file.Id, file.MimeType, err,
				)
				errChan <- &models.GdriveError{
					Err: err,
					FilePath: filepath.Join(
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
		processGdriveDlError(errChan, progress)
	}
	progress.Stop(hasErr)
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
			"gdrive error %d: could not determine file type from URL, %q",
			utils.DEV_ERROR,
			url,
		)
		utils.LogError(err, "", false, utils.ERROR)
		return "", ""
	}
	return matched[utils.GDRIVE_REGEX_ID_INDEX], fileType
}

func (gdrive *GDrive) getGdriveFileInfo(gdriveId *models.GDriveToDl, config *configs.Config) ([]*models.GdriveFileToDl, *models.GdriveError) {
	switch gdriveId.Type {
	case "file":
		fileInfo, err := gdrive.GetFileDetails(
			gdriveId,
			config,
		)
		if err != nil {
			return nil, &models.GdriveError{
				Err:      err,
				FilePath: gdriveId.FilePath,
			}
		}
		fileInfo.FilePath = gdriveId.FilePath
		return []*models.GdriveFileToDl{fileInfo}, nil
	case "folder":
		filesInfo, err := gdrive.GetNestedFolderContents(
			gdriveId.Id,
			gdriveId.FilePath,
			config,
		)
		if err != nil {
			return nil, &models.GdriveError{
				Err:      err,
				FilePath: gdriveId.FilePath,
			}
		}
		var gdriveFilesInfo []*models.GdriveFileToDl
		for _, fileInfo := range filesInfo {
			fileInfo.FilePath = gdriveId.FilePath
			gdriveFilesInfo = append(gdriveFilesInfo, fileInfo)
		}
		return gdriveFilesInfo, nil
	default:
		return nil, &models.GdriveError{
			Err: fmt.Errorf(
				"gdrive error %d: unknown Google Drive URL type, %q",
				utils.DEV_ERROR,
				gdriveId.Type,
			),
			FilePath: gdriveId.FilePath,
		}
	}
}

// Downloads multiple GDrive files based on a slice of GDrive URL strings in parallel
func (gdrive *GDrive) DownloadGdriveUrls(gdriveUrls []*request.ToDownload, config *configs.Config) error {
	if len(gdriveUrls) == 0 {
		return nil
	}

	// Retrieve the id from the url text
	var gdriveIds []*models.GDriveToDl
	for _, gdriveUrl := range gdriveUrls {
		fileId, fileType := GetFileIdAndTypeFromUrl(gdriveUrl.Url)
		if fileId != "" && fileType != "" {
			gdriveIds = append(gdriveIds, &models.GDriveToDl{
				Id:       fileId,
				Type:     fileType,
				FilePath: gdriveUrl.FilePath,
			})
		}
	}

	// Note: Can't do API calls concurrently as to avoid being blocked by Google's bot detection
	var errSlice []*models.GdriveError
	var gdriveFilesInfo []*models.GdriveFileToDl
	baseMsg := "Getting GDrive file information from GDrive ID(s) [%d/" + fmt.Sprintf("%d]...", len(gdriveIds))
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting GDrive file information from %d GDrive ID(s)!",
			len(gdriveIds),
		),
		fmt.Sprintf(
			"Something went wrong while getting GDrive file information from %d GDrive ID(s)!\nPlease refer to the generated log files for more details.",
			len(gdriveIds),
		),
		len(gdriveIds),
	)
	progress.Start()
	for _, gdriveId := range gdriveIds {
		fileInfo, err := gdrive.getGdriveFileInfo(gdriveId, config)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			gdriveFilesInfo = append(gdriveFilesInfo, fileInfo...)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		for _, err := range errSlice {
			utils.LogMessageToPath(
				censorApiKeyFromStr(err.Err.Error()),
				err.FilePath,
				utils.ERROR,
			)
		}
	}
	progress.Stop(hasErr)

	gdrive.DownloadMultipleFiles(gdriveFilesInfo, config)
	return nil
}
