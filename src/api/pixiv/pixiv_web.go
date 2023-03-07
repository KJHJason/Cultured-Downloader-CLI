package pixiv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const (
	illust = iota
	manga
	ugoira
)

// This is due to Pixiv's strict rate limiting.
//
// Without delays, the user might get 429 too many requests
// or the user's account might get suspended.
//
// Additionally, pixiv.net is protected by cloudflare, so
// to prevent the user's IP reputation from going down, delays are added.
//
// More info: https://github.com/Nandaka/PixivUtil2/issues/477
func PixivSleep() {
	time.Sleep(utils.GetRandomTime(0.5, 1.0))
}

// Process the artwork details JSON and returns a map of urls with its file path and a slice of Ugoira structures
func ProcessArtworkJson(artworkUrlsJson interface{}, artworkType int64, postDownloadDir string) ([]map[string]string, Ugoira) {
	if artworkUrlsJson == nil {
		return nil, Ugoira{}
	}

	var urlsToDownload []map[string]string
	var ugoiraDownloadInfo Ugoira
	artworkUrls := artworkUrlsJson.(map[string]interface{})["body"]
	if artworkType == ugoira {
		ugoiraMap := artworkUrls.(map[string]interface{})
		originalUrl := ugoiraMap["originalSrc"].(string)
		frameInfoMap := MapDelaysToFilename(ugoiraMap)
		ugoiraDownloadInfo = Ugoira{
			Url:      originalUrl,
			FilePath: postDownloadDir,
			Frames:   frameInfoMap,
		}
	} else {
		for _, artworkUrl := range artworkUrls.([]interface{}) {
			artworkUrlMap := artworkUrl.(map[string]interface{})["urls"].(map[string]interface{})
			originalUrl := artworkUrlMap["original"].(string)
			urlsToDownload = append(urlsToDownload, map[string]string{
				"url":      originalUrl,
				"filepath": postDownloadDir,
			})
		}
	}
	return urlsToDownload, ugoiraDownloadInfo
}

type ArtworkDetails struct {
	Body struct {
		UserName   string `json:"userName"`
		Title      string `json:"title"`
		IllustType int64  `json:"illustType"`
	}
}

// Retrieves details of an artwork ID and returns
// the folder path to download the artwork to, the JSON response, and the artwork type
func GetArtworkDetails(artworkId, downloadPath string, cookies []http.Cookie) (string, interface{}, int64, error) {
	if artworkId == "" {
		return "", nil, -1, nil
	}

	headers := GetPixivRequestHeaders()
	headers["Referer"] = GetUserUrl(artworkId)
	url := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", artworkId)
	artworkDetailsRes, err := request.CallRequest("GET", url, 15, cookies, headers, nil, false)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork details for ID %v from %s",
			utils.CONNECTION_ERROR,
			artworkId,
			url,
		)
		return "", nil, -1, err
	}

	if artworkDetailsRes == nil {
		return "", nil, -1, nil
	}
	defer artworkDetailsRes.Body.Close()

	if artworkDetailsRes.StatusCode != 200 {
		err = fmt.Errorf(
			"pixiv error %d: failed to get details for artwork ID %s due to %s response from %s",
			utils.RESPONSE_ERROR,
			artworkId,
			artworkDetailsRes.Status,
			url,
		)
		return "", nil, -1, err
	}
	var artworkDetailsJsonRes ArtworkDetails
	resBody, err := utils.ReadResBody(artworkDetailsRes)
	if err != nil {
		err = fmt.Errorf(
			"%v\ndetails: failed to read response body for Pixiv artwork ID %s",
			err,
			artworkId,
		)
		return "", nil, -1, err
	}

	err = json.Unmarshal(resBody, &artworkDetailsJsonRes)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to unmarshal artwork details for ID %s\nJSON response: %s",
			utils.JSON_ERROR,
			artworkId,
			string(resBody),
		)
		return "", nil, -1, err
	}
	artworkJsonBody := artworkDetailsJsonRes.Body
	illustratorName := artworkJsonBody.UserName
	artworkName := artworkJsonBody.Title
	artworkPostDir := utils.GetPostFolder(filepath.Join(downloadPath, utils.PIXIV_TITLE), illustratorName, artworkId, artworkName)

	artworkType := artworkJsonBody.IllustType
	switch artworkType {
	case illust, manga: // illustration or manga
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/pages", artworkId)
	case ugoira: // ugoira
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/ugoira_meta", artworkId)
	default:
		err = fmt.Errorf(
			"pixiv error %d: unsupported artwork type %d for artwork ID %s",
			utils.JSON_ERROR,
			artworkType,
			artworkId,
		)
		return "", nil, -1, err
	}

	artworkUrlsRes, err := request.CallRequest("GET", url, 15, cookies, headers, nil, false)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork URLs for ID %s from %s due to %v",
			utils.CONNECTION_ERROR,
			artworkId,
			url,
			err,
		)
		return "", nil, -1, err
	}
	defer artworkUrlsRes.Body.Close()

	if artworkUrlsRes.StatusCode != 200 {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork URLs for ID %s due to %s response from %s",
			utils.RESPONSE_ERROR,
			artworkId,
			artworkUrlsRes.Status,
			url,
		)
		return "", nil, -1, err
	}

	artworkJson, _, err := utils.LoadJsonFromResponse(artworkUrlsRes)
	if err != nil {
		return "", nil, -1, err
	}
	return artworkPostDir, artworkJson, artworkType, nil
}

// Retrieves multiple artwork details based on the given slice of artwork IDs
// and returns a map to use for downloading and a slice of Ugoira structures
func GetMultipleArtworkDetails(artworkIds []string, downloadPath string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	var errSlice []error
	var ugoiraDetails []Ugoira
	var artworkDetails []map[string]string
	lastArtworkId := artworkIds[len(artworkIds)-1]

	baseMsg := "Getting and processing artwork details from Pixiv [%d/" + fmt.Sprintf("%d]...", len(artworkIds))
	progress := spinner.New(
		spinner.JSON_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting and processing %d artwork details from Pixiv!",
			len(artworkIds),
		),
		fmt.Sprintf(
			"Something went wrong while getting and processing %d artwork details from Pixiv!\nPlease refer to the logs for more details.",
			len(artworkIds),
		),
		len(artworkIds),
	)
	progress.Start()
	for _, artworkId := range artworkIds {
		artworkFolder, jsonRes, artworkType, err := GetArtworkDetails(artworkId, downloadPath, cookies)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		} else if artworkFolder == "" || artworkType == -1 {
			progress.MsgIncrement(baseMsg)
			continue
		}

		processedJsonRes, ugoiraInfo := ProcessArtworkJson(jsonRes, artworkType, artworkFolder)
		if artworkType == ugoira {
			ugoiraDetails = append(ugoiraDetails, ugoiraInfo)
		} else {
			artworkDetails = append(artworkDetails, processedJsonRes...)
		}

		progress.MsgIncrement(baseMsg)
		if artworkId != lastArtworkId {
			PixivSleep()
		}
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, errSlice...)
	}
	progress.Stop(hasErr)

	return artworkDetails, ugoiraDetails
}

// Query Pixiv's API for all the illustrator's posts
func GetIllustratorPosts(illustratorId, artworkType string, cookies []http.Cookie) ([]string, error) {
	artworkType = strings.ToLower(artworkType)
	headers := GetPixivRequestHeaders()
	headers["Referer"] = GetIllustUrl(illustratorId)
	url := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/profile/all", illustratorId)

	res, err := request.CallRequest("GET", url, 5, cookies, headers, nil, false)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get illustrator's posts with an ID of %s due to %v",
			utils.CONNECTION_ERROR,
			illustratorId,
			err,
		)
		return nil, err
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		err = fmt.Errorf(
			"pixiv error %d: failed to get illustrator's posts with an ID of %s due to %s response",
			utils.RESPONSE_ERROR,
			illustratorId,
			res.Status,
		)
		return nil, err
	}

	jsonBody, _, err := utils.LoadJsonFromResponse(res)
	if err != nil {
		return nil, err
	}

	jsonBody = jsonBody.(map[string]interface{})["body"]

	var artworkIds []string
	if artworkType == "all" || artworkType == "illust_and_ugoira" {
		illusts := jsonBody.(map[string]interface{})["illusts"]
		if reflect.TypeOf(illusts).Kind() == reflect.Map {
			for illustId := range illusts.(map[string]interface{}) {
				artworkIds = append(artworkIds, illustId)
			}
		}
	}

	if artworkType == "all" || artworkType == "manga" {
		manga := jsonBody.(map[string]interface{})["manga"]
		if reflect.TypeOf(manga).Kind() == reflect.Map {
			for mangaId := range manga.(map[string]interface{}) {
				artworkIds = append(artworkIds, mangaId)
			}
		}
	}
	return artworkIds, nil
}

// Get posts from multiple illustrators and returns a map and a slice for Ugoira structures for downloads
func GetMultipleIllustratorPosts(illustratorIds []string, downloadPath, artworkType string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	var errSlice []error
	var artworkIdsSlice []string

	baseMsg := "Getting artwork details from illustrator(s) on Pixiv [%d/" + fmt.Sprintf("%d]...", len(illustratorIds))
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting artwork details from %d illustrator(s) on Pixiv!",
			len(illustratorIds),
		),
		fmt.Sprintf(
			"Something went wrong while getting artwork details from %d illustrator(s) on Pixiv!\nPlease refer to the logs for more details.",
			len(illustratorIds),
		),
		len(illustratorIds),
	)
	progress.Start()
	for _, illustratorId := range illustratorIds {
		artworkIds, err := GetIllustratorPosts(illustratorId, artworkType, cookies)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			artworkIdsSlice = append(artworkIdsSlice, artworkIds...)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, errSlice...)
	}
	progress.Stop(hasErr)

	artworksSlice, ugoiraSlice := GetMultipleArtworkDetails(
		artworkIdsSlice, 
		downloadPath, 
		cookies,
	)
	return artworksSlice, ugoiraSlice
}

type PixivTag struct {
	Body struct {
		IllustManga struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
		} `json:"illustManga"`
	} `json:"body"`
}

// Process the tag search results JSON and returns a slice of artwork IDs
func ProcessTagJsonResults(res *http.Response) ([]string, error) {
	var pixivTagJson PixivTag
	resBody, err := utils.ReadResBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resBody, &pixivTagJson)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to unmarshal json for Pixiv's Tag JSON due to %v\nJSON: %s",
			utils.JSON_ERROR,
			err,
			string(resBody),
		)
		return nil, err
	}

	artworksSlice := []string{}
	for _, illust := range pixivTagJson.Body.IllustManga.Data {
		artworksSlice = append(artworksSlice, illust.Id)
	}
	return artworksSlice, nil
}

// Query Pixiv's API and search for posts based on the supplied tag name
// which will return a map and a slice of Ugoira structures for downloads
func TagSearch(tagName, downloadPath string, dlOptions *PixivDlOptions, minPage, maxPage int, cookies []http.Cookie) ([]map[string]string, []Ugoira, bool) {
	url := "https://www.pixiv.net/ajax/search/artworks/" + tagName
	params := map[string]string{
		"word":   tagName,              // search term
		"s_mode": dlOptions.SearchMode, // search mode: s_tag, s_tag_full, s_tc
		"order":  dlOptions.SortOrder,  // sort order: date, popular, popular_male, popular_female
		// (add "_d" suffix for descending order, e.g. date_d)
		"mode": dlOptions.RatingMode,  //  r18, safe, or all for both
		"type": dlOptions.ArtworkType, // illust_and_ugoira, manga, all
	}

	if minPage > maxPage {
		minPage, maxPage = maxPage, minPage
	}

	var errSlice []error
	var artworkIds []string
	headers := GetPixivRequestHeaders()
	headers["Referer"] = fmt.Sprintf("https://www.pixiv.net/tags/%s/artworks", tagName)
	for page := minPage; page <= maxPage; page++ {
		params["p"] = strconv.Itoa(page) // page number
		res, err := request.CallRequest("GET", url, 5, cookies, headers, params, true)
		if err != nil {
			err = fmt.Errorf(
				"pixiv error %d: failed to get tag search results for %s due to %v",
				utils.CONNECTION_ERROR,
				tagName,
				err,
			)
			errSlice = append(errSlice, err)
			continue
		}

		tagArtworkIds, err := ProcessTagJsonResults(res)
		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}

		artworkIds = append(artworkIds, tagArtworkIds...)
		if page != maxPage {
			PixivSleep()
		}
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, errSlice...)
	}

	artworkSlice, ugoiraSlice := GetMultipleArtworkDetails(artworkIds, downloadPath, cookies)
	return artworkSlice, ugoiraSlice, hasErr
}
