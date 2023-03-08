package request

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"regexp"

	"github.com/fatih/color"
	"github.com/quic-go/quic-go/http3"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	// Since the URLs below will be redirected to Fantia's AWS S3 URL, 
	// we need to use HTTP/2 as it is not supported by HTTP/3 yet.
	FANTIA_ALBUM_URL = regexp.MustCompile(
		`^https://fantia.jp/posts/[\d]+/album_image`,
	)
	FANTIA_DOWNLOAD_URL = regexp.MustCompile(
		`^https://fantia.jp/posts/[\d]+/download/[\d]+`,
	)

	HTTP3_SUPPORT_ARR = [4]string{
		"https://www.pixiv.net",
		"https://app-api.pixiv.net",

		"https://www.google.com",
		"https://drive.google.com",
	}
)

func getHttp2Client(disableCompression bool) *http.Client {
	if disableCompression {
		return &http.Client{
			Transport: &http.Transport{
				DisableCompression: true,
			},
		}
	}
	return &http.Client{}
}

func getHttp3Client(disableCompression bool) *http.Client {
	if disableCompression {
		return &http.Client{
			Transport: &http3.RoundTripper{
				DisableCompression: true,
			},
		}
	}
	return &http.Client{
		Transport: &http3.RoundTripper{},
	}
}

// Get a new HTTP/2 or HTTP/3 client based on the given URL if it's supported
func getHttpClient(reqUrl *string, disableCompression bool) *http.Client {
	if FANTIA_DOWNLOAD_URL.MatchString(*reqUrl) || FANTIA_ALBUM_URL.MatchString(*reqUrl) {
		return getHttp2Client(disableCompression)
	}

	// check if the URL supports HTTP/3 first
	// before falling back to the default HTTP/2.
	for _, domain := range HTTP3_SUPPORT_ARR {
		if strings.HasPrefix(*reqUrl, domain) {
			return getHttp3Client(disableCompression)
		}
	}
	return getHttp2Client(disableCompression)
}

// send the request to the target URL and retries if the request was not successful
func sendRequest(reqUrl string, req *http.Request, timeout int, checkStatus, disableCompression bool) (*http.Response, error) {
	var err error
	var res *http.Response

	client := getHttpClient(&reqUrl, disableCompression)
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
}

	errMsg := fmt.Sprintf(
		"the request to %s failed after %d retries",
		reqUrl,
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

// Check for active internet connection (To be used at the start of the program)
func CheckInternetConnection() {
	_, err := CallRequest(
		"HEAD",
		"https://www.google.com",
		10,
		nil,
		nil,
		nil,
		false,
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
	fileUrl,
	filePath string,
	cookies []http.Cookie,
	headers,
	params map[string]string,
	overwriteExistingFiles bool,
) error {
	// Send a HEAD request first to get the expected file size from the Content-Length header.
	// A GET request might work but most of the time
	// as the Content-Length header may not present due to chunked encoding.
	headRes, err := CallRequestNoCompression("HEAD", fileUrl, 10, cookies, headers, params, true)
	if err != nil {
		return err
	}
	fileReqContentLength := headRes.ContentLength
	headRes.Body.Close()

	downloadTimeout := 25 * 60 // 25 minutes in seconds as downloads
	// can take quite a while for large files (especially for Pixiv)
	// However, the average max file size on these platforms is around 300MB.
	// Note: Fantia do have a max file size per post of 3GB if one paid extra for it.
	res, err := CallRequest("GET", fileUrl, downloadTimeout, cookies, headers, params, true)
	if err != nil {
		err = fmt.Errorf(
			"error %d: failed to download file, more info => %v\nurl: %s",
			utils.DOWNLOAD_ERROR,
			err,
			fileUrl,
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
		filename = utils.GetLastPartOfURL(filename)
		filenameWithoutExt := utils.RemoveExtFromFilename(filename)
		filePath = filepath.Join(filePath, filenameWithoutExt+strings.ToLower(filepath.Ext(filename)))
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
		errorMsg := fmt.Sprintf("failed to download %s due to %v", fileUrl, err)
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
	urls *[]map[string]string,
	maxConcurrency int,
	cookies []http.Cookie,
	headers,
	params map[string]string,
	overwriteExistingFiles bool,
) {
	urlsLen := len(*urls)
	if urlsLen == 0 {
		return
	}
	if urlsLen < maxConcurrency {
		maxConcurrency = urlsLen
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
	for _, url := range *urls {
		wg.Add(1)
		queue <- struct{}{}
		go func(fileUrl, filePath string) {
			defer wg.Done()
			err := DownloadURL(
				fileUrl,
				filePath,
				cookies,
				headers,
				params,
				overwriteExistingFiles,
			)
			if err != nil {
				errChan <- fmt.Errorf(
					"failed to download url, \"%s\", to \"%s\", please refer to the error details below:\n%v",
					fileUrl,
					filePath,
					err,
				)
			}
			progress.MsgIncrement(baseMsg)
			<-queue
		}(url["url"], url["filepath"])
	}
	close(queue)
	wg.Wait()
	close(errChan)

	hasErr := false
	if len(errChan) > 0 {
		hasErr = true
		utils.LogErrors(false, &errChan)
	}
	progress.Stop(hasErr)
}
