package request

import (
	"context"
	"errors"
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
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
	"github.com/quic-go/quic-go/http3"
)

// Get a new HTTP/2 or HTTP/3 client based on the request arguments
func GetHttpClient(reqArgs *RequestArgs) *http.Client {
	if reqArgs.Http2 {
		return &http.Client{
			Transport: &http.Transport{
				DisableCompression: reqArgs.DisableCompression,
			},
		}
	}
	return &http.Client{
		Transport: &http3.RoundTripper{
			DisableCompression: reqArgs.DisableCompression,
		},
	}
}

// add headers to the request
func AddHeaders(headers map[string]string, defaultUserAgent string, req *http.Request) {
	if len(headers) == 0 {
		return
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Add(
			"User-Agent", defaultUserAgent,
		)
	}
}

// add cookies to the request
func AddCookies(reqUrl string, cookies []*http.Cookie, req *http.Request) {
	if len(cookies) == 0 {
		return
	}

	for _, cookie := range cookies {
		if strings.Contains(reqUrl, cookie.Domain) {
			req.AddCookie(cookie)
		}
	}
}

// add params to the request
func AddParams(params map[string]string, req *http.Request) {
	if len(params) == 0 {
		return
	}

	query := req.URL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()
}

// send the request to the target URL and retries if the request was not successful
func sendRequest(req *http.Request, reqArgs *RequestArgs) (*http.Response, error) {
	AddCookies(reqArgs.Url, reqArgs.Cookies, req)
	AddHeaders(reqArgs.Headers, reqArgs.UserAgent, req)
	AddParams(reqArgs.Params, req)

	var err error
	var res *http.Response

	client := GetHttpClient(reqArgs)
	client.Timeout = time.Duration(reqArgs.Timeout) * time.Second
	for i := 1; i <= utils.RETRY_COUNTER; i++ {
		res, err = client.Do(req)
		if err == nil {
			if !reqArgs.CheckStatus {
				return res, nil
			} else if res.StatusCode == 200 {
				return res, nil
			}
			res.Body.Close()
		} else if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}

		if i < utils.RETRY_COUNTER {
			time.Sleep(utils.GetRandomDelay())
		}
	}

	errMsg := fmt.Sprintf(
		"the request to %s failed after %d retries",
		reqArgs.Url,
		utils.RETRY_COUNTER,
	)
	if err != nil {
		err = fmt.Errorf("%s, more info => %v",
			errMsg,
			err,
		)
	} else {
		err = errors.New(errMsg)
	}
	return nil, err
}

// CallRequest is used to make a request to a URL and return the response
//
// If the request fails, it will retry the request again up
// to the defined max retries in the constants.go in utils package
func CallRequest(reqArgs *RequestArgs) (*http.Response, error) {
	reqArgs.ValidateArgs()
	req, err := http.NewRequestWithContext(
		reqArgs.Context,
		reqArgs.Method,
		reqArgs.Url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error %d: unable to create a new request, more info => %v",
			utils.DEV_ERROR,
			err,
		)
	}

	return sendRequest(req, reqArgs)
}

// Check for active internet connection (To be used at the start of the program)
func CheckInternetConnection() {
	_, err := CallRequest(
		&RequestArgs{
			Url:         "https://www.google.com",
			Method:      "HEAD",
			Timeout:     10,
			CheckStatus: false,
			Http3:       true,
		},
	)
	if err != nil {
		color.Red(
			fmt.Sprintf(
				"error %d: unable to connect to the internet, more info => %v",
				utils.DEV_ERROR,
				err,
			),
		)
		os.Exit(1)
	}
}

// Sends a request with the given data
func CallRequestWithData(reqArgs *RequestArgs, data map[string]string) (*http.Response, error) {
	reqArgs.ValidateArgs()
	form := url.Values{}
	for key, value := range data {
		form.Add(key, value)
	}
	if len(data) > 0 {
		reqArgs.Headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequestWithContext(
		reqArgs.Context,
		reqArgs.Method,
		reqArgs.Url,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}

	return sendRequest(req, reqArgs)
}

type RequestHandler func (reqArgs *RequestArgs) (*http.Response, error)

// DownloadUrlWithHandler is used to download a file from a URL
//
// Note: If the file already exists, the download process will be skipped
func DownloadUrlWithHandler(filePath string, queue chan struct{}, reqArgs *RequestArgs, overwriteExistingFile bool, reqHandler RequestHandler) error {
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
	headRes, err := reqHandler(
		&RequestArgs{
			Url:         reqArgs.Url,
			Method:      "HEAD",
			Timeout:     10,
			Cookies:     reqArgs.Cookies,
			Headers:     reqArgs.Headers,
			CheckStatus: true,
			Http3:       reqArgs.Http3,
			Http2:       reqArgs.Http2,
		},
	)
	if err != nil {
		return err
	}
	fileReqContentLength := headRes.ContentLength
	headRes.Body.Close()

	reqArgs.Context = ctx
	res, err := reqHandler(reqArgs)
	if err != nil {
		if err == context.Canceled {
			return err
		}

		err = fmt.Errorf(
			"error %d: failed to download file, more info => %v\nurl: %s",
			utils.DOWNLOAD_ERROR,
			err,
			reqArgs.Url,
		)
		return err
	}
	defer res.Body.Close()

	// check if filepath already have a filename attached
	if filepath.Ext(filePath) == "" {
		os.MkdirAll(filePath, 0666)
		filename, err := url.PathUnescape(res.Request.URL.String())
		if err != nil {
			// should never happen but just in case
			err = fmt.Errorf(
				"error %d: failed to unescape URL, more info => %v\nurl: %s",
				utils.UNEXPECTED_ERROR,
				err,
				res.Request.URL.String(),
			)
			return err
		}
		filename = utils.GetLastPartOfUrl(filename)
		filenameWithoutExt := utils.RemoveExtFromFilename(filename)
		filePath = filepath.Join(
			filePath,
			filenameWithoutExt+strings.ToLower(filepath.Ext(filename)),
		)
	} else {
		filePathDir := filepath.Dir(filePath)
		os.MkdirAll(filePathDir, 0666)
		filePathWithoutExt := utils.RemoveExtFromFilename(filePath)
		filePath = filePathWithoutExt + strings.ToLower(filepath.Ext(filePath))
	}

	// check if the file size matches the content length
	// if not, then the file does not exist or is corrupted and should be re-downloaded
	fileSize, _ := utils.GetFileSize(filePath)
	if fileReqContentLength != -1 {
		if fileSize == fileReqContentLength {
			// If the file already exists and the file size
			// matches the expected file size in the Content-Length header,
			// then skip the download process.
			return nil
		}
	} else {
		if !overwriteExistingFile && fileSize > 0 {
			// If the file already exists and have more than 0 bytes
			// but the Content-Length header does not exist in the response,
			// we will assume that the file is already downloaded
			// and skip the download process if the overwrite flag is false.
			return nil
		}
	}

	file, err := os.Create(filePath) // create the file
	if err != nil {
		err = fmt.Errorf(
			"error %d: failed to create file, more info => %v\nfile path: %s",
			utils.OS_ERROR,
			err,
			filePath,
		)
		return err
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

		if err == context.Canceled {
			return err
		}
		errorMsg := fmt.Sprintf("failed to download %s due to %v", reqArgs.Url, err)
		utils.LogError(err, errorMsg, false, utils.ERROR)
		return nil
	}
	file.Close()
	return nil
}

// DownloadUrl is the same as DownloadUrlWithHandler but uses the default request handler (CallRequest)
func DownloadUrl(filePath string, queue chan struct{}, reqArgs *RequestArgs, overwriteExistingFile bool) error {
	return DownloadUrlWithHandler(filePath, queue, reqArgs, overwriteExistingFile, CallRequest)
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
				<-queue
				wg.Done()
			}()
			err := DownloadUrlWithHandler(
				filePath,
				queue,
				&RequestArgs{
					Url:       fileUrl,
					Method:    "GET",
					Timeout:   utils.DOWNLOAD_TIMEOUT,
					Cookies:   dlOptions.Cookies,
					Headers:   dlOptions.Headers,
					Http2:     !dlOptions.UseHttp3,
					Http3:     dlOptions.UseHttp3,
					UserAgent: config.UserAgent,
				},
				config.OverwriteFiles,
				reqHandler,
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