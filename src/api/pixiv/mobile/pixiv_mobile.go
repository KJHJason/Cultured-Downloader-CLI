package pixivmobile

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

type PixivMobile struct {
	// API information and its endpoints
	baseUrl      string
	clientId     string
	clientSecret string
	userAgent    string
	authTokenUrl string
	loginUrl     string
	redirectUri  string
	refreshToken string

	// User given arguments
	apiTimeout int

	// Access token information
	accessTokenMu  sync.Mutex
	accessTokenMap accessTokenInfo
}

// Get a new PixivMobile structure
func NewPixivMobile(refreshToken string, timeout int) *PixivMobile {
	pixivMobile := &PixivMobile{
		baseUrl:       utils.PIXIV_MOBILE_URL,
		clientId:      "MOBrBDS8blbauoSck0ZfDbtuzpyT",
		clientSecret:  "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
		userAgent:     "PixivIOSApp/7.13.3 (iOS 14.6; iPhone13,2)",
		authTokenUrl:  "https://oauth.secure.pixiv.net/auth/token",
		loginUrl:      utils.PIXIV_MOBILE_URL + "/web/v1/login",
		redirectUri:   utils.PIXIV_MOBILE_URL + "/web/v1/users/auth/pixiv/callback",
		refreshToken:  refreshToken,
		apiTimeout:    timeout,
	}
	if refreshToken != "" {
		// refresh the access token and verify it
		err := pixivMobile.refreshAccessToken()
		if err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}
	}
	return pixivMobile
}

// This is due to Pixiv's strict rate limiting.
//
// Without delays, the user might get 429 too many requests
// or the user's account might get suspended.
//
// Additionally, pixiv.net is protected by cloudflare, so
// to prevent the user's IP reputation from going down, delays are added.
func (pixiv *PixivMobile) Sleep() {
	time.Sleep(utils.GetRandomTime(1.0, 1.5))
}

// Get the required headers to communicate with the Pixiv API
func (pixiv *PixivMobile) getHeaders(additional map[string]string) map[string]string {
	headers := make(map[string]string)
	for k, v := range additional {
		headers[k] = v
	}

	baseHeaders := map[string]string{
		"User-Agent":     pixiv.userAgent,
		"App-OS":         "ios",
		"App-OS-Version": "14.6",
		"Authorization":  "Bearer " + pixiv.accessTokenMap.accessToken,
	}
	for k, v := range baseHeaders {
		headers[k] = v
	}
	return headers
}


// Sends a request to the Pixiv API and refreshes the access token if required
//
// Returns the JSON interface and errors if any
func (pixiv *PixivMobile) SendRequest(reqArgs *request.RequestArgs) (*http.Response, error) {
	if reqArgs.Method == "" {
		reqArgs.Method = "GET"
	}
	if reqArgs.Timeout == 0 {
		reqArgs.Timeout = pixiv.apiTimeout
	}
	useHttp3 := utils.IsHttp3Supported(utils.PIXIV_MOBILE, true)
	reqArgs.Http3 = useHttp3
	reqArgs.Http2 = !useHttp3
	reqArgs.ValidateArgs()

	req, err := http.NewRequest(reqArgs.Method, reqArgs.Url, nil)
	if err != nil {
		return nil, err
	}

	refreshed, err := pixiv.refreshTokenIfReq()
	if err != nil {
		return nil, err
	}

	for k, v := range pixiv.getHeaders(reqArgs.Headers) {
		req.Header.Set(k, v)
	}
	request.AddParams(reqArgs.Params, req)

	var res *http.Response
	client := request.GetHttpClient(reqArgs)
	client.Timeout = time.Duration(reqArgs.Timeout) * time.Second
	for i := 1; i <= utils.RETRY_COUNTER; i++ {
		res, err = client.Do(req)
		if err == nil {
			if refreshed {
				continue
			} else if res.StatusCode == 200 || !reqArgs.CheckStatus {
				return res, nil
			}
		}
		time.Sleep(utils.GetRandomDelay())
	}
	return nil, fmt.Errorf(
		"request to %s failed after %d retries",
		reqArgs.Url,
		utils.RETRY_COUNTER,
	)
}
