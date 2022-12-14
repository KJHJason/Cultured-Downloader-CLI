package pixiv

import (
	"fmt"
	"os"
	"time"
	"regexp"
	"strconv"
	"strings"
	"net/http"
	"path/filepath"
	"crypto/sha256"
	"encoding/base64"
	CryptoRand "crypto/rand"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
)

type PixivMobile struct {
	baseUrl string
	clientId string
	clientSecret string
	userAgent string
	authTokenUrl string
	loginUrl string
	redirectUri string
	refreshToken string
	accessToken  string
	apiTimeout int
}

// Get a new PixivMobile structure
func NewPixivMobile(refreshToken string, timeout int) *PixivMobile {
	pixivMobile := &PixivMobile{
		baseUrl: "https://app-api.pixiv.net",
		clientId: "MOBrBDS8blbauoSck0ZfDbtuzpyT",
		clientSecret: "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj",
		userAgent: "PixivIOSApp/7.13.3 (iOS 14.6; iPhone13,2)",
		authTokenUrl: "https://oauth.secure.pixiv.net/auth/token",
		loginUrl: "https://app-api.pixiv.net/web/v1/login",
		redirectUri: "https://app-api.pixiv.net/web/v1/users/auth/pixiv/callback",
		refreshToken: refreshToken,
		accessToken: "",
		apiTimeout: timeout,
	}
	if refreshToken != "" {
		// refresh the access token and verify it
		pixivMobile.RefreshAccessToken()
	}
	return pixivMobile
}

// Convert the page number to the offset as one page will have 30 illustrations instead of the usual 60.
func ConvertPageNumToOffset(minPageNum, maxPageNum int) (int, int) {
	// Check for negative page numbers
	if minPageNum < 0 {
		minPageNum = 1
	}
	if maxPageNum < 0 {
		maxPageNum = 1
	}

	// Swap the page numbers if the min is greater than the max
	if minPageNum > maxPageNum {
		minPageNum, maxPageNum = maxPageNum, minPageNum
	}

	minOffset := 60 * (minPageNum - 1)
	maxOffset := 60 * (maxPageNum - minPageNum + 1)
	// Check if the offset is larger than Pixiv's max offset
	if maxOffset > 5000 {
		maxOffset = 5000
	}
	if minOffset > 5000 {
		minOffset = 5000
	}
	return minOffset, maxOffset
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
func (pixiv *PixivMobile) GetHeaders(additional... map[string]string) map[string]string {
	headers := map[string]string{
		"User-Agent": pixiv.userAgent,
		"App-OS": "ios",
		"App-OS-Version": "14.6",
		"Authorization": "Bearer " + pixiv.accessToken,
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
func (pixiv *PixivMobile) RefreshAccessToken() {
	headers := map[string]string{"User-Agent": pixiv.userAgent}
	data := map[string]string{
		"client_id": pixiv.clientId,
		"client_secret": pixiv.clientSecret,
		"grant_type": "refresh_token",
		"include_policy": "true",
		"refresh_token": pixiv.refreshToken,
	}
	res, err := request.CallRequestWithData(
		pixiv.authTokenUrl, "POST", 
		pixiv.apiTimeout, nil, data, headers, nil, false,
	)
	if err != nil || res.StatusCode != 200 {
		if err == nil {
			res.Body.Close()
		}
		color.Red("Pixiv: Failed to refresh token...")
		color.Red("Please check your refresh token and try again or use the \"-pixiv_start_oauth\" flag to get a new refresh token.")
		os.Exit(1)
	}

	resJson, _ := utils.LoadJsonFromResponse(res)
	pixiv.accessToken = resJson.(map[string]interface{})["access_token"].(string)
}

// Reads the response JSON and checks if the access token has expired, 
// if so, refreshes the access token for future requests.
//
// Returns a boolean indicating if the access token was refreshed.
func (pixiv *PixivMobile) RefreshTokenIfReq(resJson interface{}, statusCode int) bool {
	if statusCode != 400 {
		return false
	}

	errJsonMap := resJson.(map[string]interface{})["error"].(map[string]interface{})
	message := errJsonMap["message"].(string)
	if !strings.Contains(message, "invalid_grant") {
		return false
	}

	pixiv.RefreshAccessToken()
	return true
}

// Sends a request to the Pixiv API and refreshes the access token if required
//
// Returns the JSON interface and errors if any
func (pixiv *PixivMobile) SendRequest(
	reqUrl string, additionalHeaders, params map[string]string, checkStatus bool,
) (interface{}, error) {
	req, err := http.NewRequest("GET", reqUrl, nil)
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
			jsonRes, _ := utils.LoadJsonFromResponse(res)
			if pixiv.RefreshTokenIfReq(jsonRes, res.StatusCode) {
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
	utils.LogError(nil, "Pixiv: " + err.Error(), false)
	return nil, err
}

// Start the OAuth flow to get the refresh token
func (pixiv *PixivMobile) StartOauthFlow() {
	codeVerifierBytes := make([]byte, 32)
	_, err := CryptoRand.Read(codeVerifierBytes)
	if err != nil {
		panic(err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(codeVerifierBytes)
	codeChallenge := S256([]byte(codeVerifier))

	loginParams := map[string]string{
		"code_challenge": codeChallenge,
		"code_challenge_method": "S256",
		"client": "pixiv-android",
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
	codeRegex := regexp.MustCompile(`^[\w-]{43}$`)
	headers := map[string]string{"User-Agent": "PixivAndroidApp/5.0.234 (Android 11; Pixel 5)"}
	for {
		var code string
		fmt.Print(color.YellowString("Please enter the code you received from Pixiv: "))
		_, err := fmt.Scanln(&code)
		fmt.Println()
		if err != nil {
			color.Red("Failed to read inputted code: " + err.Error())
			continue
		}
		if !codeRegex.MatchString(code) {
			color.Red("Invalid code format...")
			continue
		}

		data := map[string]string{
			"client_id": pixiv.clientId,
			"client_secret": pixiv.clientSecret,
			"code": code,
			"code_verifier": codeVerifier,
			"grant_type": "authorization_code",
			"include_policy": "true",
			"redirect_uri": pixiv.redirectUri,
		}
		res, err := request.CallRequestWithData(
			pixiv.authTokenUrl, "POST", 
			pixiv.apiTimeout, nil, data, headers, nil, true,
		)
		if err != nil {
			color.Red("Please check if the code you entered is correct.")
			continue
		} 

		resJson, _ := utils.LoadJsonFromResponse(res)
		refreshToken := resJson.(map[string]interface{})["refresh_token"].(string)
		color.Green("Your Pixiv Refresh Token: " + refreshToken)
		color.Yellow("Please save your refresh token somewhere SECURE and do NOT share it with anyone!")
		return
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
		Url: ugoiraDlUrl,
		Frames: frameInfoMap,
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
		filepath.Join(downloadPath, api.PixivTitle), illustratorName, artworkId, artworkTitle,
	)

	if artworkType == "ugoira" {
		return []map[string]string{{
			"artwork_id": artworkId,
			"filepath": artworkFolderPath,
		}}
	}

	singlePageImageUrl := artworkJson["meta_single_page"].(map[string]interface{})["original_image_url"] 
	isSinglePage := singlePageImageUrl != nil
	if isSinglePage {
		imageUrl := singlePageImageUrl.(string)
		artworksToDownload = append(artworksToDownload, map[string]string{
			"url": imageUrl,
			"filepath": artworkFolderPath,
		})
	} else {
		images := artworkJson["meta_pages"].([]interface{})
		for _, image := range images {
			imageUrl := image.(map[string]interface{})["image_urls"].(map[string]interface{})["original"].(string)
			artworksToDownload = append(artworksToDownload, map[string]string{
				"url": imageUrl,
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
func (pixiv *PixivMobile) CheckForUgoira(artworksToDownload *[]map[string]string) ([]map[string]string, []Ugoira) {
	var filteredArtworks []map[string]string
	var ugoiraArr []Ugoira
	lastIdx := len(*artworksToDownload) - 1
	for idx, artwork := range (*artworksToDownload) {
		if _, ok := artwork["artwork_id"]; ok {
			ugoiraInfo := pixiv.GetUgoiraMetadata(artwork["artwork_id"], artwork["filepath"])
			if idx != lastIdx {
				pixiv.Sleep()
			}
			ugoiraArr = append(ugoiraArr, ugoiraInfo)
		} else {
			filteredArtworks = append(filteredArtworks, artwork)
		}
	}
	return filteredArtworks, ugoiraArr
}

// Query Pixiv's API (mobile) to get the JSON of an artwork ID
func (pixiv *PixivMobile) GetArtworkDetails(artworkId, downloadPath string) ([]map[string]string, []Ugoira) {
	artworkUrl := pixiv.baseUrl + "/v1/illust/detail"
	params := map[string]string{"illust_id": artworkId}
	res, err := pixiv.SendRequest(artworkUrl, pixiv.GetHeaders(), params, true)
	if err != nil {
		utils.LogError(nil, "Pixiv: Failed to get artwork details for " + artworkId, false)
	}
	artworkDetails := pixiv.ProcessArtworkJson(
		res.(map[string]interface{})["illust"].(map[string]interface{}), downloadPath,
	)
	artworkDetails, ugoiraArr := pixiv.CheckForUgoira(&artworkDetails)
	return artworkDetails, ugoiraArr
}

func (pixiv *PixivMobile) GetMultipleArtworkDetails(artworkIds []string, downloadPath string) ([]map[string]string, []Ugoira) {
	var artworksToDownload []map[string]string
	var ugoiraArr []Ugoira
	lastIdx := len(artworkIds) - 1
	bar := utils.GetProgressBar(
		len(artworkIds), 
		"Getting and processing artwork details...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting and processing %d artwork details!", len(artworkIds))),
	)
	for idx, artworkId := range artworkIds {
		artworkDetails, ugoiraInfo := pixiv.GetArtworkDetails(artworkId, downloadPath)
		artworksToDownload = append(artworksToDownload, artworkDetails...)
		ugoiraArr = append(ugoiraArr, ugoiraInfo...)
		if idx != lastIdx {
			pixiv.Sleep()
		}
		bar.Add(1)
	}
	return artworksToDownload, ugoiraArr
}

// Query Pixiv's API (mobile) to get all the posts JSON(s) of a user ID
func (pixiv *PixivMobile) GetIllustratorPosts(userId, downloadPath, artworkType string) ([]map[string]string, []Ugoira) {
	params := map[string]string{
		"user_id": userId,
		"filter": "for_ios",
	}
	if artworkType == "illust_and_ugoira" || artworkType == "all" {
		params["type"] = "illust"
	} else {
		params["type"] = "manga"
	}

	var artworksToDownload []map[string]string
	nextUrl := pixiv.baseUrl + "/v1/user/illusts"
startLoop:
	for nextUrl != "" {
		res, err := pixiv.SendRequest(nextUrl, pixiv.GetHeaders(), params, true)
		if err != nil {
			utils.LogError(nil, "Pixiv: Failed to get illustrator posts for " + userId, false)
			return nil, nil
		}
		resJson := res.(map[string]interface{})
		artworksToDownload = append(artworksToDownload, pixiv.ProcessMultipleArtworkJson(resJson, downloadPath)...)

		jsonNextUrl := resJson["next_url"]
		if jsonNextUrl == nil {
			nextUrl = ""
		} else {
			nextUrl = jsonNextUrl.(string)
			pixiv.Sleep()
		}
	}
	if params["type"] == "illust" && artworkType == "all" {
		params["type"] = "manga"
		nextUrl = pixiv.baseUrl + "/v1/user/illusts"
		goto startLoop
	}

	artworksToDownload, ugoiraArr := pixiv.CheckForUgoira(&artworksToDownload)
	return artworksToDownload, ugoiraArr
}

func (pixiv *PixivMobile) GetMultipleIllustratorPosts(userIds []string, downloadPath, artworkType string) ([]map[string]string, []Ugoira) {
	var artworksToDownload []map[string]string
	var ugoiraArr []Ugoira
	lastIdx := len(userIds) - 1
	bar := utils.GetProgressBar(
		len(userIds), 
		"Getting artwork details from illustrator(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting artwork details from %d illustrator(s)!", len(userIds))),
	)
	for idx, userId := range userIds {
		artworkDetails, ugoiraInfo := pixiv.GetIllustratorPosts(userId, downloadPath, artworkType)
		artworksToDownload = append(artworksToDownload, artworkDetails...)
		ugoiraArr = append(ugoiraArr, ugoiraInfo...)
		if idx != lastIdx {
			pixiv.Sleep()
		}
		bar.Add(1)
	}
	return artworksToDownload, ugoiraArr
}

// Query Pixiv's API (mobile) to get the JSON of a search query
func (pixiv *PixivMobile) TagSearch(tagName, searchTarget, sortOrder, downloadPath string, minPage, maxPage int) ([]map[string]string, []Ugoira) {
	var artworksToDownload []map[string]string
	nextUrl := pixiv.baseUrl + "/v1/search/illust"
	minOffset, maxOffset := ConvertPageNumToOffset(minPage, maxPage)

	// Convert searchTarget to the correct value
	// based on the Pixiv's ajax web API
	searchTargetMap := map[string]string{
		"s_tag": "partial_match_for_tags",
		"s_tag_full": "exact_match_for_tags",
		"s_tc": "title_and_caption",
	}
	searchTarget, ok := searchTargetMap[searchTarget]
	if !ok {
		panic("Pixiv: Invalid search target")
	}

	// Convert sortOrder to the correct value
	// based on the Pixiv's ajax web API
	if strings.Contains(sortOrder, "popular") {
		sortOrder = "popular_desc" // only supports popular_desc
	} else if sortOrder == "date_d" {
		sortOrder = "date_desc"
	} else {
		sortOrder = "date_asc"
	}
	params := map[string]string{
		"word": tagName,
		"search_target": searchTarget,
		"sort": sortOrder,
		"filter": "for_ios",
		"offset": strconv.Itoa(minOffset),
	}
	for nextUrl != "" && minOffset != maxOffset {
		res, err := pixiv.SendRequest(nextUrl, pixiv.GetHeaders(), params, true)
		if err != nil {
			utils.LogError(nil, "Pixiv: Failed to search for " + tagName, false)
			return nil, nil
		}
		resJson := res.(map[string]interface{})
		artworksToDownload = append(artworksToDownload, pixiv.ProcessMultipleArtworkJson(resJson, downloadPath)...)

		minOffset += 30
		params["offset"] = strconv.Itoa(minOffset)
		jsonNextUrl := resJson["next_url"]
		if jsonNextUrl == nil {
			nextUrl = ""
		} else {
			nextUrl = jsonNextUrl.(string)
			pixiv.Sleep()
		}
	}

	artworksToDownload, ugoiraArr := pixiv.CheckForUgoira(&artworksToDownload)
	return artworksToDownload, ugoiraArr
}