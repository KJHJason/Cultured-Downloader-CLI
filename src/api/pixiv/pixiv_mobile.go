package pixiv

import (
	CryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
	"github.com/pkg/browser"
)

type accessTokenInfo struct {
	accessToken string    // The access token that will be used to communicate with the Pixiv's Mobile API
	expiresAt   time.Time // The time when the access token expires
}

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
		accessTokenMu: sync.Mutex{},
		refreshToken:  refreshToken,
		apiTimeout:    timeout,
	}
	if refreshToken != "" {
		// refresh the access token and verify it
		err := pixivMobile.RefreshAccessToken()
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
func (pixiv *PixivMobile) GetHeaders(additional ...map[string]string) map[string]string {
	headers := map[string]string{
		"User-Agent":     pixiv.userAgent,
		"App-OS":         "ios",
		"App-OS-Version": "14.6",
		"Authorization":  "Bearer " + pixiv.accessTokenMap.accessToken,
	}
	for _, header := range additional {
		for k, v := range header {
			headers[k] = v
		}
	}
	return headers
}

// Perform a S256 transformation method on a byte array
func S256(bytes []byte) string {
	hash := sha256.Sum256(bytes)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// Refresh the access token
func (pixiv *PixivMobile) RefreshAccessToken() error {
	pixiv.accessTokenMu.Lock()
	defer pixiv.accessTokenMu.Unlock()

	res, err := request.CallRequestWithData(
		&request.RequestArgs{
			Url:       pixiv.authTokenUrl,
			Method:    "POST",
			Timeout:   pixiv.apiTimeout,
			UserAgent: pixiv.userAgent,
		},
		map[string]string{
			"client_id":      pixiv.clientId,
			"client_secret":  pixiv.clientSecret,
			"grant_type":     "refresh_token",
			"include_policy": "true",
			"refresh_token":  pixiv.refreshToken,
		},
	)
	if err != nil || res.StatusCode != 200 {
		const errPrefix = "pixiv mobile error"
		if err == nil {
			res.Body.Close()
			err = fmt.Errorf(
				"%s %d: failed to refresh token due to %s response from Pixiv\n"+
					"Please check your refresh token and try again or use the \"-pixiv_start_oauth\" flag to get a new refresh token",
				errPrefix,
				utils.RESPONSE_ERROR,
				res.Status,
			)
		} else {
			err = fmt.Errorf(
				"%s %d: failed to refresh token due to %v\n"+
					"Please check your internet connection and try again",
				errPrefix,
				utils.CONNECTION_ERROR,
				err,
			)
		}
		return err
	}

	resJson, _, err := utils.LoadJsonFromResponse(res)
	if err != nil {
		return err
	}

	expiresIn := resJson.(map[string]interface{})["expires_in"].(float64) - 15 // usually 3600 but minus 15 seconds to be safe
	pixiv.accessTokenMap.accessToken = resJson.(map[string]interface{})["access_token"].(string)
	pixiv.accessTokenMap.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return nil
}

// Reads the response JSON and checks if the access token has expired,
// if so, refreshes the access token for future requests.
//
// Returns a boolean indicating if the access token was refreshed.
func (pixiv *PixivMobile) RefreshTokenIfReq() (bool, error) {
	if pixiv.accessTokenMap.accessToken != "" && pixiv.accessTokenMap.expiresAt.After(time.Now()) {
		return false, nil
	}

	err := pixiv.RefreshAccessToken()
	if err != nil {
		return true, err
	}
	return true, nil
}

// Sends a request to the Pixiv API and refreshes the access token if required
//
// Returns the JSON interface and errors if any
func (pixiv *PixivMobile) SendRequest(reqUrl string, additionalHeaders, params map[string]string, checkStatus bool) (interface{}, error) {
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return nil, err
	}

	refreshed, err := pixiv.RefreshTokenIfReq()
	if err != nil {
		return nil, err
	}

	for k, v := range pixiv.GetHeaders(additionalHeaders) {
		req.Header.Set(k, v)
	}
	request.AddParams(params, req)

	var res *http.Response
	client := &http.Client{}
	client.Timeout = time.Duration(pixiv.apiTimeout) * time.Second
	for i := 1; i <= utils.RETRY_COUNTER; i++ {
		res, err = client.Do(req)
		if err == nil {
			jsonRes, _, err := utils.LoadJsonFromResponse(res)
			if err != nil && i == utils.RETRY_COUNTER {
				return nil, err
			}

			if refreshed {
				continue
			} else if !checkStatus {
				return jsonRes, nil
			} else if res.StatusCode == 200 {
				return jsonRes, nil
			}
		}
		time.Sleep(utils.GetRandomDelay())
	}
	err = fmt.Errorf("request to %s failed after %d retries", reqUrl, utils.RETRY_COUNTER)
	return nil, err
}

var pixivOauthCodeRegex = regexp.MustCompile(`^[\w-]{43}$`)

// Start the OAuth flow to get the refresh token
func (pixiv *PixivMobile) StartOauthFlow() error {
	// create a random 32 bytes that is cryptographically secure
	codeVerifierBytes := make([]byte, 32)
	_, err := CryptoRand.Read(codeVerifierBytes)
	if err != nil {
		// should never happen but just in case
		err = fmt.Errorf(
			"pixiv mobile error %d: failed to generate random bytes, more info => %v",
			utils.DEV_ERROR,
			err,
		)
		return err
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(codeVerifierBytes)
	codeChallenge := S256([]byte(codeVerifier))

	loginParams := map[string]string{
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
		"client":                "pixiv-android",
	}

	loginUrl := pixiv.loginUrl + "?" + utils.ParamsToString(loginParams)
	err = browser.OpenURL(loginUrl)
	if err != nil {
		color.Red("Pixiv: Failed to open browser: " + err.Error())
		color.Red("Please open the following URL in your browser:")
		color.Red(loginUrl)
	} else {
		color.Green("Opened a new tab in your browser to\n" + loginUrl)
	}

	color.Yellow("If unsure, follow the guide below:")
	color.Yellow("https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/pixiv_oauth_guide.md\n")
	for {
		var code string
		fmt.Print(color.YellowString("Please enter the code you received from Pixiv: "))
		_, err := fmt.Scanln(&code)
		fmt.Println()
		if err != nil {
			color.Red("Failed to read inputted code: " + err.Error())
			continue
		}
		if !pixivOauthCodeRegex.MatchString(code) {
			color.Red("Invalid code format...")
			continue
		}

		res, err := request.CallRequestWithData(
			&request.RequestArgs{
				Url:         pixiv.authTokenUrl,
				Method:      "POST",
				Timeout:     pixiv.apiTimeout,
				CheckStatus: true,
				UserAgent:   "PixivAndroidApp/5.0.234 (Android 11; Pixel 5)",
			},
			map[string]string{
				"client_id":      pixiv.clientId,
				"client_secret":  pixiv.clientSecret,
				"code":           code,
				"code_verifier":  codeVerifier,
				"grant_type":     "authorization_code",
				"include_policy": "true",
				"redirect_uri":   pixiv.redirectUri,
			},
		)
		if err != nil {
			color.Red("Please check if the code you entered is correct.")
			continue
		}

		resJson, _, err := utils.LoadJsonFromResponse(res)
		if err != nil {
			color.Red(err.Error())
			continue
		}

		refreshToken := resJson.(map[string]interface{})["refresh_token"].(string)
		color.Green("Your Pixiv Refresh Token: " + refreshToken)
		color.Yellow("Please save your refresh token somewhere SECURE and do NOT share it with anyone!")
		return nil
	}
}

// Returns the Ugoira structure with the necessary information to download the ugoira
func (pixiv *PixivMobile) GetUgoiraMetadata(illustId, postDownloadDir string) Ugoira {
	ugoiraUrl := pixiv.baseUrl + "/v1/ugoira/metadata"
	params := map[string]string{"illust_id": illustId}
	additionalHeaders := pixiv.GetHeaders(
		map[string]string{"Referer": pixiv.baseUrl},
	)
	res, err := pixiv.SendRequest(ugoiraUrl, additionalHeaders, params, true)
	if err != nil {
		errMsg := "Pixiv: Failed to get ugoira metadata for " + illustId
		utils.LogMessageToPath(errMsg, postDownloadDir)
	}

	resJson := res.(map[string]interface{})["ugoira_metadata"].(map[string]interface{})
	ugoiraDlUrl := resJson["zip_urls"].(map[string]interface{})["medium"].(string)
	ugoiraDlUrl = strings.Replace(ugoiraDlUrl, "600x600", "1920x1080", 1)

	// map the files to their delay
	frameInfoMap := MapDelaysToFilename(resJson)
	return Ugoira{
		Url:      ugoiraDlUrl,
		Frames:   frameInfoMap,
		FilePath: postDownloadDir,
	}
}

// Process the artwork JSON and returns a slice of map that contains the urls of the images and the file path
func (pixiv *PixivMobile) ProcessArtworkJson(artworkJson map[string]interface{}, downloadPath string) []map[string]string {
	var artworksToDownload []map[string]string
	artworkId := fmt.Sprintf("%d", int64(artworkJson["id"].(float64)))
	artworkTitle := artworkJson["title"].(string)
	artworkType := artworkJson["type"].(string)
	illustratorJson := artworkJson["user"].(map[string]interface{})
	illustratorName := illustratorJson["name"].(string)
	artworkFolderPath := utils.GetPostFolder(
		filepath.Join(downloadPath, utils.PIXIV_TITLE), illustratorName, artworkId, artworkTitle,
	)

	if artworkType == "ugoira" {
		return []map[string]string{{
			"artwork_id": artworkId,
			"filepath":   artworkFolderPath,
		}}
	}

	singlePageImageUrl := artworkJson["meta_single_page"].(map[string]interface{})["original_image_url"]
	isSinglePage := singlePageImageUrl != nil
	if isSinglePage {
		imageUrl := singlePageImageUrl.(string)
		artworksToDownload = append(artworksToDownload, map[string]string{
			"url":      imageUrl,
			"filepath": artworkFolderPath,
		})
	} else {
		images := artworkJson["meta_pages"].([]interface{})
		for _, image := range images {
			imageUrl := image.(map[string]interface{})["image_urls"].(map[string]interface{})["original"].(string)
			artworksToDownload = append(artworksToDownload, map[string]string{
				"url":      imageUrl,
				"filepath": artworkFolderPath,
			})
		}
	}
	return artworksToDownload
}

// The same as the ProcessArtworkJson function but for mutliple JSONs at once
// (Those with the "illusts" key which holds a slice of maps containing the artwork JSON)
func (pixiv *PixivMobile) ProcessMultipleArtworkJson(resJson map[string]interface{}, downloadPath string) []map[string]string {
	artworksMap := resJson["illusts"]
	if artworksMap == nil {
		return nil
	}

	var artworksToDownload []map[string]string
	for _, artwork := range artworksMap.([]interface{}) {
		artworkJson := artwork.(map[string]interface{})
		artworksToDownload = append(artworksToDownload, pixiv.ProcessArtworkJson(artworkJson, downloadPath)...)
	}
	return artworksToDownload
}

// Checks the processed JSON results for a Ugoira type artwork, if found, make a call to Pixiv's API (Mobile)
// and get its metadata (the URL to download the ugoira from and its frames' delay)
//
// Returns the filtered slice of map that contains the artworks details to download and a slice of Ugoira structures
func (pixiv *PixivMobile) CheckForUgoira(artworksToDownload []map[string]string) ([]map[string]string, []*Ugoira) {
	var filteredArtworks []map[string]string
	var ugoiraSlice []*Ugoira
	lastIdx := len(artworksToDownload) - 1
	for idx, artwork := range artworksToDownload {
		if _, ok := artwork["artwork_id"]; ok {
			ugoiraInfo := pixiv.GetUgoiraMetadata(artwork["artwork_id"], artwork["filepath"])
			if idx != lastIdx {
				pixiv.Sleep()
			}
			ugoiraSlice = append(ugoiraSlice, &ugoiraInfo)
		} else {
			filteredArtworks = append(filteredArtworks, artwork)
		}
	}
	return filteredArtworks, ugoiraSlice
}

// Query Pixiv's API (mobile) to get the JSON of an artwork ID
func (pixiv *PixivMobile) GetArtworkDetails(artworkId, downloadPath string) ([]map[string]string, []*Ugoira, error) {
	artworkUrl := pixiv.baseUrl + "/v1/illust/detail"
	params := map[string]string{"illust_id": artworkId}

	res, err := pixiv.SendRequest(
		artworkUrl,
		pixiv.GetHeaders(),
		params,
		true,
	)
	if err != nil {
		err = fmt.Errorf(
			"pixiv mobile error %d: failed to get artwork details for %s, more info => %v",
			utils.CONNECTION_ERROR,
			artworkId,
			err,
		)
		return nil, nil, err
	}

	artworkDetails := pixiv.ProcessArtworkJson(
		res.(map[string]interface{})["illust"].(map[string]interface{}),
		downloadPath,
	)
	artworkDetails, ugoiraSlice := pixiv.CheckForUgoira(artworkDetails)
	return artworkDetails, ugoiraSlice, nil
}

func (pixiv *PixivMobile) getMultipleArtworkDetails(artworkIds []string, downloadPath string) ([]map[string]string, []*Ugoira) {
	var artworksToDownload []map[string]string
	var ugoiraSlice []*Ugoira
	artworkIdsLen := len(artworkIds)
	lastIdx := artworkIdsLen - 1

	var errSlice []error
	baseMsg := "Getting and processing artwork details from Pixiv's Mobile API [%d/" + fmt.Sprintf("%d]...", artworkIdsLen)
	progress := spinner.New(
		spinner.JSON_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting and processing %d artwork details from Pixiv's Mobile API!",
			artworkIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting and processing %d artwork details from Pixiv's Mobile API!\nPlease refer to the logs for more details.",
			artworkIdsLen,
		),
		artworkIdsLen,
	)
	progress.Start()
	for idx, artworkId := range artworkIds {
		artworkDetails, ugoiraInfo, err := pixiv.GetArtworkDetails(artworkId, downloadPath)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		}

		artworksToDownload = append(artworksToDownload, artworkDetails...)
		ugoiraSlice = append(ugoiraSlice, ugoiraInfo...)
		if idx != lastIdx {
			pixiv.Sleep()
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, errSlice...)
	}
	progress.Stop(hasErr)

	return artworksToDownload, ugoiraSlice
}

// Query Pixiv's API (mobile) to get all the posts JSON(s) of a user ID
func (pixiv *PixivMobile) GetIllustratorPosts(userId, pageNum, downloadPath, artworkType string) ([]map[string]string, []*Ugoira, error) {
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, nil, err
	}
	minOffset, maxOffset := ConvertPageNumToOffset(minPage, maxPage, false)

	params := map[string]string{
		"user_id": userId,
		"filter":  "for_ios",
		"offset":  strconv.Itoa(minOffset),
	}
	if artworkType == "all" {
		params["type"] = "illust"
	} else {
		params["type"] = artworkType
	}

	var artworksToDownload []map[string]string
	nextUrl := pixiv.baseUrl + "/v1/user/illusts"

startLoop:
	curOffset := minOffset
	for nextUrl != "" {
		res, err := pixiv.SendRequest(
			nextUrl,
			pixiv.GetHeaders(),
			params,
			true,
		)
		if err != nil {
			err = fmt.Errorf(
				"pixiv mobile error %d: failed to get illustrator posts for %s, more info => %v",
				utils.CONNECTION_ERROR,
				userId,
				err,
			)
			return nil, nil, err
		}
		resJson := res.(map[string]interface{})
		artworksToDownload = append(
			artworksToDownload,
			pixiv.ProcessMultipleArtworkJson(resJson, downloadPath)...,
		)

		curOffset += 30
		params["offset"] = strconv.Itoa(curOffset)
		jsonNextUrl := resJson["next_url"]
		if jsonNextUrl == nil || (hasMax && curOffset >= maxOffset) {
			nextUrl = ""
		} else {
			nextUrl = jsonNextUrl.(string)
			pixiv.Sleep()
		}
	}

	if params["type"] == "illust" && artworkType == "all" {
		// if the user is downloading both
		// illust and manga, loop again to get the manga
		params["type"] = "manga"
		nextUrl = pixiv.baseUrl + "/v1/user/illusts"
		goto startLoop
	}

	artworksToDownload, ugoiraSlice := pixiv.CheckForUgoira(artworksToDownload)
	return artworksToDownload, ugoiraSlice, nil
}

func (pixiv *PixivMobile) getMultipleIllustratorPosts(userIds, pageNums []string, downloadPath, artworkType string) ([]map[string]string, []*Ugoira) {
	var artworksToDownload []map[string]string
	var ugoiraSlice []*Ugoira
	userIdsLen := len(userIds)
	lastIdx := userIdsLen - 1

	var errSlice []error
	baseMsg := "Getting artwork details from illustrator(s) on Pixiv [%d/" + fmt.Sprintf("%d]...", userIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting artwork details from %d illustrator(s) on Pixiv!",
			userIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting artwork details from %d illustrator(s) on Pixiv!\nPlease refer to the logs for more details.",
			userIdsLen,
		),
		userIdsLen,
	)
	progress.Start()
	for idx, userId := range userIds {
		artworkDetails, ugoiraInfo, err := pixiv.GetIllustratorPosts(
			userId, 
			pageNums[idx],
			downloadPath, 
			artworkType,
		)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		}

		artworksToDownload = append(artworksToDownload, artworkDetails...)
		ugoiraSlice = append(ugoiraSlice, ugoiraInfo...)
		if idx != lastIdx {
			pixiv.Sleep()
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, errSlice...)
	}
	progress.Stop(hasErr)

	return artworksToDownload, ugoiraSlice
}

// Query Pixiv's API (mobile) to get the JSON of a search query
func (pixiv *PixivMobile) tagSearch(tagName, downloadPath, pageNum string, dlOptions *PixivDlOptions) ([]map[string]string, []*Ugoira, bool) {
	var artworksToDownload []map[string]string
	nextUrl := pixiv.baseUrl + "/v1/search/illust"
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		utils.LogError(
			err,
			"",
			false,
		)
		return nil, nil, true
	}
	minOffset, maxOffset := ConvertPageNumToOffset(minPage, maxPage, false)

	params := map[string]string{
		"word":          tagName,
		"search_target": dlOptions.SearchMode,
		"sort":          dlOptions.SortOrder,
		"filter":        "for_ios",
		"offset":        strconv.Itoa(minOffset),
	}
	curOffset := minOffset
	for nextUrl != "" {
		res, err := pixiv.SendRequest(nextUrl, pixiv.GetHeaders(), params, true)
		if err != nil {
			utils.LogError(
				fmt.Errorf(
					"pixiv mobile error %d: failed to search for %s, more info => %v",
					utils.CONNECTION_ERROR,
					tagName,
					err,
				),
				"",
				false,
			)
			return nil, nil, true
		}
		resJson := res.(map[string]interface{})
		artworksToDownload = append(
			artworksToDownload, 
			pixiv.ProcessMultipleArtworkJson(resJson, downloadPath)...,
		)

		curOffset += 30
		params["offset"] = strconv.Itoa(curOffset)
		jsonNextUrl := resJson["next_url"]
		if jsonNextUrl == nil || (hasMax && curOffset >= maxOffset){
			nextUrl = ""
		} else {
			nextUrl = jsonNextUrl.(string)
			pixiv.Sleep()
		}
	}

	artworksToDownload, ugoiraSlice := pixiv.CheckForUgoira(artworksToDownload)
	return artworksToDownload, ugoiraSlice, false
}
