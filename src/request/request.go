package request

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// send the request to the target URL and retries if the request was not successful
func sendRequest(reqUrl string, req *http.Request, timeout int, checkStatus, disableCompression bool) (*http.Response, error) {
	var err error
	var res *http.Response

	transport := &http.Transport{
		DisableCompression: disableCompression,
	}
	client := &http.Client{
		Transport: transport,
	}
	client.Timeout = time.Duration(timeout) * time.Second
	for i := 1; i <= utils.RETRY_COUNTER; i++ {
		res, err = client.Do(req)
		if err == nil {
			if !checkStatus {
				return res, nil
			} else if res.StatusCode == 200 {
				return res, nil
			}
			res.Body.Close()
		}

		if i < utils.RETRY_COUNTER {
			time.Sleep(utils.GetRandomDelay())
		}
	}
	err = fmt.Errorf("the request to %s failed after %d retries", reqUrl, utils.RETRY_COUNTER)
	return nil, err
}

// add headers to the request
func AddHeaders(headers map[string]string, req *http.Request) {
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Add(
			"User-Agent", utils.USER_AGENT,
		)
	}
}

// add cookies to the request
func AddCookies(reqUrl string, cookies []http.Cookie, req *http.Request) {
	for _, cookie := range cookies {
		if strings.Contains(reqUrl, cookie.Domain) {
			req.AddCookie(&cookie)
		}
	}
}

// add params to the request
func AddParams(params map[string]string, req *http.Request) {
	if len(params) > 0 {
		query := req.URL.Query()
		for key, value := range params {
			query.Add(key, value)
		}
		req.URL.RawQuery = query.Encode()
	}
}

// CallRequest is used to make a request to a URL and return the response
//
// If the request fails, it will retry the request again up 
// to the defined max retries in the constants.go in utils package
func CallRequest(
	method, 
	reqUrl string, 
	timeout int, 
	cookies []http.Cookie, 
	additionalHeaders, 
	params map[string]string, 
	checkStatus bool,
) (*http.Response, error) {
	req, err := http.NewRequest(method, reqUrl, nil)
	if err != nil {
		return nil, err
	}

	AddCookies(reqUrl, cookies, req)
	AddHeaders(additionalHeaders, req)
	AddParams(params, req)
	return sendRequest(reqUrl, req, timeout, checkStatus, false)
}

// Sends a request with the given data
func CallRequestWithData(
	reqUrl, 
	method string, 
	timeout int, 
	cookies []http.Cookie, 
	data, 
	additionalHeaders, 
	params map[string]string, 
	checkStatus bool,
) (*http.Response, error) {
	form := url.Values{}
	for key, value := range data {
		form.Add(key, value)
	}
	if len(data) > 0 {
		additionalHeaders["Content-Type"] = "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequest(method, reqUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	AddCookies(reqUrl, cookies, req)
	AddHeaders(additionalHeaders, req)
	AddParams(params, req)
	return sendRequest(reqUrl, req, timeout, checkStatus, false)
}

// Sends a request to the given URL but disables golang's HTTP client compression
//
// Useful for calling a HEAD request to obtain the actual uncompressed file's file size
func CallRequestNoCompression(
	method, reqUrl string, timeout int, cookies []http.Cookie, 
	additionalHeaders, params map[string]string, checkStatus bool,
) (*http.Response, error) {
	req, err := http.NewRequest(method, reqUrl, nil)
	if err != nil {
		return nil, err
	}

	AddCookies(reqUrl, cookies, req)
	AddHeaders(additionalHeaders, req)
	AddParams(params, req)
	return sendRequest(reqUrl, req, timeout, checkStatus, true)
}

// DownloadURL is used to download a file from a URL
//
// Note: If the file already exists, the download process will be skipped
func DownloadURL(
	fileURL, 
	filePath string, 
	cookies []http.Cookie, 
	headers, 
	params map[string]string, 
	overwriteExistingFiles bool,
) error {
	// Send a HEAD request first to get the expected file size from the Content-Length header.
	// A GET request might work but most of the time 
	// as the Content-Length header may not present due to chunked encoding.
	headRes, err := CallRequestNoCompression("HEAD", fileURL, 10, cookies, headers, params, true)
	if err != nil {
		return err
	}
	fileReqContentLength := headRes.ContentLength
	headRes.Body.Close()

	downloadTimeout := 25 * 60  // 25 minutes in seconds as downloads 
								// can take quite a while for large files (especially for Pixiv)
								// However, the average max file size on these platforms is around 300MB.
								// Note: Fantia do have a max file size per post of 3GB if one paid extra for it.
	res, err := CallRequest("GET", fileURL, downloadTimeout, cookies, headers, params, true)
	if err != nil {
		err = fmt.Errorf(
			"error %d: failed to download file, more info => %v\nurl: %s",
			utils.DOWNLOAD_ERROR,
			err,
			fileURL,
		)
		return err
	}
	defer res.Body.Close()

	// check if filepath already have a filename attached
	if filepath.Ext(filePath) == "" {
		os.MkdirAll(filePath, 0755)
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
		filename = utils.GetLastPartOfURL(filename)
		filenameWithoutExt := utils.RemoveExtFromFilename(filename)
		filePath = filepath.Join(filePath, filenameWithoutExt + strings.ToLower(filepath.Ext(filename)))
	} else {
		filePathDir := filepath.Dir(filePath)
		os.MkdirAll(filePathDir, 0755)
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
		if !overwriteExistingFiles && fileSize > 0 {
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
		os.Remove(filePath)
		errorMsg := fmt.Sprintf("failed to download %s due to %v", fileURL, err)
		utils.LogError(err, errorMsg, false)
		return nil
	}
	file.Close()
	return nil
}

// DownloadURLsParallel is used to download multiple files from URLs in parallel
//
// Note: If the file already exists, the download process will be skipped
func DownloadURLsParallel(
	urls []map[string]string, 
	maxConcurrency int, 
	cookies []http.Cookie, 
	headers, 
	params map[string]string, 
	overwriteExistingFiles bool,
) {
	if len(urls) == 0 {
		return
	}
	if len(urls) < maxConcurrency {
		maxConcurrency = len(urls)
	}

	bar := utils.GetProgressBar(
		len(urls),
		"Downloading...",
		utils.GetCompletionFunc(
			fmt.Sprintf("Downloaded %d files", len(urls)),
		),
	)
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	errChan := make(chan error, len(urls))
	for _, url := range urls {
		wg.Add(1)
		queue <- struct{}{}
		go func(fileUrl, filePath string) {
			defer wg.Done()
			err := DownloadURL(fileUrl, filePath, cookies, headers, params, overwriteExistingFiles)
			if err != nil {
				errChan <- fmt.Errorf(
					"failed to download url, \"%s\", to \"%s\", please refer to the error details below:\n%v",
					fileUrl,
					filePath,
					err,
				)
			}
			bar.Add(1)
			<-queue
		}(url["url"], url["filepath"])
	}
	close(queue)
	wg.Wait()
	close(errChan)
	utils.LogErrors(false, &errChan)
}