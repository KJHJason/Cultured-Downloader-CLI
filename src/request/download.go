package request

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func getFullFilePath(res *http.Response, filePath string) (string, error) {
	// check if filepath already have a filename attached
	if filepath.Ext(filePath) != "" {
		filePathDir := filepath.Dir(filePath)
		os.MkdirAll(filePathDir, 0666)
		filePathWithoutExt := utils.RemoveExtFromFilename(filePath)
		return filePathWithoutExt + strings.ToLower(filepath.Ext(filePath)), nil
	}

	os.MkdirAll(filePath, 0666)
	filename, err := url.PathUnescape(res.Request.URL.String())
	if err != nil {
		// should never happen but just in case
		return "", fmt.Errorf(
			"error %d: failed to unescape URL, more info => %v\nurl: %s",
			utils.UNEXPECTED_ERROR,
			err,
			res.Request.URL.String(),
		)
	}
	filename = utils.GetLastPartOfUrl(filename)
	filenameWithoutExt := utils.RemoveExtFromFilename(filename)
	filePath = filepath.Join(
		filePath,
		filenameWithoutExt + strings.ToLower(filepath.Ext(filename)),
	)
	return filePath, nil
}

// check if the file size matches the content length
// if not, then the file does not exist or is corrupted and should be re-downloaded
func checkIfCanSkipDl(contentLength int64, filePath string, forceOverwrite bool) bool {
	fileSize, err := utils.GetFileSize(filePath)
	if err != nil {
		if err != os.ErrNotExist {
			// if the error wasn't because the file does not exist,
			// then log the error and continue with the download process
			utils.LogError(err, "", false, utils.ERROR)
		}
		return false
	}

	if fileSize == contentLength {
		// If the file already exists and the file size
		// matches the expected file size in the Content-Length header,
		// then skip the download process.
		return true
	} else if !forceOverwrite && fileSize > 0 {
		// If the file already exists and have more than 0 bytes
		// but the Content-Length header does not exist in the response,
		// we will assume that the file is already downloaded
		// and skip the download process if the overwrite flag is false.
		return true
	}
	return false
}

func DlToFile(res *http.Response, url, filePath string) error {
	file, err := os.Create(filePath) // create the file
	if err != nil {
		return fmt.Errorf(
			"error %d: failed to create file, more info => %v\nfile path: %s",
			utils.OS_ERROR,
			err,
			filePath,
		)
	}

	// write the body to file
	// https://stackoverflow.com/a/11693049/16377492
	_, err = io.Copy(file, res.Body)
	if err != nil {
		file.Close()
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

		if err != context.Canceled {
			errorMsg := fmt.Sprintf("failed to download %s due to %v", url, err)
			utils.LogError(err, errorMsg, false, utils.ERROR)
			err = nil
		}
		return err
	}
	file.Close()
	return nil
}

// DownloadUrl is used to download a file from a URL
//
// Note: If the file already exists, the download process will be skipped
func DownloadUrl(filePath string, queue chan struct{}, reqArgs *RequestArgs, overwriteExistingFile bool) error {
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
	// Send a HEAD request first to get the expected file size from the Content-Length header.
	// A GET request might work but most of the time
	// as the Content-Length header may not present due to chunked encoding.
	headRes, err := reqArgs.RequestHandler(
		&RequestArgs{
			Url:         reqArgs.Url,
			Method:      "HEAD",
			Timeout:     10,
			Cookies:     reqArgs.Cookies,
			Headers:     reqArgs.Headers,
			UserAgent:   reqArgs.UserAgent,
			CheckStatus: true,
			Http3:       reqArgs.Http3,
			Http2:       reqArgs.Http2,
			Context:     ctx,
		},
	)
	if err != nil {
		return err
	}
	fileReqContentLength := headRes.ContentLength
	headRes.Body.Close()

	reqArgs.Context = ctx
	res, err := reqArgs.RequestHandler(reqArgs)
	if err != nil {
		if err != context.Canceled {
			err = fmt.Errorf(
				"error %d: failed to download file, more info => %v\nurl: %s",
				utils.DOWNLOAD_ERROR,
				err,
				reqArgs.Url,
			)
		}
		return err
	}
	defer res.Body.Close()

	filePath, err = getFullFilePath(res, filePath)
	if err != nil {
		return err
	}

	if !checkIfCanSkipDl(fileReqContentLength, filePath, overwriteExistingFile) {
		err = DlToFile(res, reqArgs.Url, filePath)
	}
	return err
}

// DownloadUrls is used to download multiple files from URLs concurrently
//
// Note: If the file already exists, the download process will be skipped
func DownloadUrlsWithHandler(urlInfoSlice []*ToDownload, dlOptions *DlOptions, config *configs.Config, reqHandler RequestHandler) {
	urlsLen := len(urlInfoSlice)
	if urlsLen == 0 {
		return
	}
	if urlsLen < dlOptions.MaxConcurrency {
		dlOptions.MaxConcurrency = urlsLen
	}

	var wg sync.WaitGroup
	queue := make(chan struct{}, dlOptions.MaxConcurrency)
	errChan := make(chan error, urlsLen)

	baseMsg := "Downloading files [%d/" + fmt.Sprintf("%d]...", urlsLen)
	progress := spinner.New(
		spinner.DL_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished downloading %d files",
			urlsLen,
		),
		fmt.Sprintf(
			"Something went wrong while downloading %d files.\nPlease refer to the logs for more details.",
			urlsLen,
		),
		urlsLen,
	)
	progress.Start()
	for _, urlInfo := range urlInfoSlice {
		wg.Add(1)
		go func(fileUrl, filePath string) {
			defer func() {
				wg.Done()
				<-queue
			}()
			err := DownloadUrl(
				filePath,
				queue,
				&RequestArgs{
					Url:            fileUrl,
					Method:         "GET",
					Timeout:        utils.DOWNLOAD_TIMEOUT,
					Cookies:        dlOptions.Cookies,
					Headers:        dlOptions.Headers,
					Http2:          !dlOptions.UseHttp3,
					Http3:          dlOptions.UseHttp3,
					UserAgent:      config.UserAgent,
					RequestHandler: reqHandler,
				},
				config.OverwriteFiles,
			)
			if err != nil {
				errChan <- err
			}

			if err != context.Canceled {
				progress.MsgIncrement(baseMsg)
			}
		}(urlInfo.Url, urlInfo.FilePath)
	}
	wg.Wait()
	close(queue)
	close(errChan)

	hasErr := false
	if len(errChan) > 0 {
		hasErr = true
		if kill := utils.LogErrors(false, errChan, utils.ERROR); kill {
			progress.KillProgram(
				"Stopped downloading files (incomplete downloads will be deleted)...",
			)
		}
	}
	progress.Stop(hasErr)
}

// Same as DownloadUrlsWithHandler but uses the default request handler (CallRequest)
func DownloadUrls(urlInfoSlice []*ToDownload, dlOptions *DlOptions, config *configs.Config) {
	DownloadUrlsWithHandler(urlInfoSlice, dlOptions, config, CallRequest)
}
