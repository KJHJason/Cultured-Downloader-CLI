package request

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"strconv"
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
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

	if userAgent, ok := headers["User-Agent"]; !ok || userAgent == ""{
		headers["User-Agent"] = defaultUserAgent
	}

	for key, value := range headers {
		req.Header.Add(key, value)
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
		} else {
			break
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
	} else if res != nil {
		err = fmt.Errorf("%s, status code => %s",
			errMsg,
			res.Status,
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

type versionInfo struct {
	Major int
	Minor int
	Patch int
}

func processVer(apiResVer string) (*versionInfo, error) {
	// split the version string by "."
	ver := strings.Split(apiResVer, ".")
	if len(ver) != 3 {
		return nil, fmt.Errorf(
			"github error %d: unable to process the latest version, %q",
			utils.DEV_ERROR,
			apiResVer,
		)
	}

	// convert the version string to int
	verSlice := make([]int, 3)
	for i, v := range ver {
		verInt, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf(
				"github error %d: unable to process the latest version, %q",
				utils.DEV_ERROR,
				apiResVer,
			)
		}
		verSlice[i] = verInt
	}

	return &versionInfo{
		Major: verSlice[0],
		Minor: verSlice[1],
		Patch: verSlice[2],
	}, nil
}

type GithubApiRes struct {
	TagName string `json:"tag_name"`
	HtmlUrl string `json:"html_url"`
}

// check for the latest version of the program
func CheckVer() error {
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		"Checking for the latest version...",
		"",
		"Failed to check for the latest version, please refer to the logs for more details...",
		0,
	)
	url := "https://api.github.com/repos/KJHJason/Cultured-Downloader-CLI/releases/latest"
	res, err := CallRequest(
		&RequestArgs{
			Url:         url,
			Method:      "GET",
			Timeout:     5,
			CheckStatus: false,
			Http3:       false,
			Http2:       true,
		},
	)
	progress.Start()
	if err != nil || res.StatusCode != 200 {
		errMsg := fmt.Sprintf(
			"github error %d: unable to check for the latest version",
			utils.CONNECTION_ERROR,
		)
		if err != nil {
			errMsg += fmt.Sprintf(", more info => %v", err)
		}

		progress.Stop(true)
		return errors.New(errMsg)
	}

	var apiRes GithubApiRes
	if err := utils.LoadJsonFromResponse(res, &apiRes); err != nil {
		errMsg := fmt.Sprintf(
			"github error %d: unable to marshal the response from the API into an interface",
			utils.UNEXPECTED_ERROR,
		)
		progress.Stop(true)
		return errors.New(errMsg)
	}

	latestVer, err := processVer(apiRes.TagName)
	if err != nil {
		errMsg := fmt.Sprintf(
			"github error %d: unable to process the latest version",
			utils.UNEXPECTED_ERROR,
		)
		progress.Stop(true)
		return errors.New(errMsg)
	}

	programVer, err := processVer(utils.VERSION)
	if err != nil {
		errMsg := fmt.Sprintf(
			"error %d: unable to process the program version",
			utils.DEV_ERROR,
		)
		panic(errMsg)
	}

	// check if the latest version is greater than the current version
	// well, I do hate nested if statements, but if it works, it works.
	var outdated bool
	if latestVer.Major > programVer.Major {
		outdated = true
	} else if latestVer.Major == programVer.Major {
		if latestVer.Minor > programVer.Minor {
			outdated = true
		} else if latestVer.Minor == programVer.Minor {
			if latestVer.Patch > programVer.Patch {
				outdated = true
			}
		}
	}

	if outdated {
		progress.ErrMsg = fmt.Sprintf(
			"Warning: this program is outdated, the latest version %q is available at %s",
			apiRes.TagName,
			apiRes.HtmlUrl,
		)
	} else {
		progress.SuccessMsg = "This program is up to date!"
	}
	progress.Stop(outdated)
	return nil
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
