package pixiv

import (
	"encoding/json"
	"fmt"
	"time"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"reflect"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
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
			Url: originalUrl,
			FilePath: postDownloadDir,
			Frames: frameInfoMap,
		}
	} else {
		for _, artworkUrl := range artworkUrls.([]interface{}) {
			artworkUrlMap := artworkUrl.(map[string]interface{})["urls"].(map[string]interface{})
			originalUrl := artworkUrlMap["original"].(string)
			urlsToDownload = append(urlsToDownload, map[string]string{
				"url": originalUrl,
				"filepath": postDownloadDir,
			})
		}
	}
	return urlsToDownload, ugoiraDownloadInfo
}

type ArtworkDetails struct {
	Body struct {
		UserName string `json:"userName"`
		Title string `json:"title"`
		IllustType int64 `json:"illustType"`
	}
}

// Retrieves details of an artwork ID and returns 
// the folder path to download the artwork to, the JSON response, and the artwork type
func GetArtworkDetails(artworkId, downloadPath string, cookies []http.Cookie) (string, interface{}, int64) {
	if artworkId == "" {
		return "", nil, -1
	}

	headers := GetPixivRequestHeaders()
	headers["Referer"] = GetUserUrl(artworkId)
	url := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", artworkId)
	artworkDetailsRes, err := request.CallRequest("GET", url, 15, cookies, headers, nil, false)
	if err != nil {
		errorMsg := fmt.Sprintf("Pixiv: Failed to get artwork details for %v\nURL: %s", artworkId, url)
		utils.LogError(err, errorMsg, false)
		return "", nil, -1
	}
	if artworkDetailsRes == nil {
		return "", nil, -1
	}
	defer artworkDetailsRes.Body.Close()
	if artworkDetailsRes.StatusCode != 200 {
		errorMsg := fmt.Sprintf(
			"Pixiv: Failed to get details for artwork ID %s due to %s response\nURL: %s",
			artworkId,
			artworkDetailsRes.Status,
			url,
		)
		utils.LogError(nil, errorMsg, false)
		return "", nil, -1
	}
	var artworkDetailsJsonRes ArtworkDetails
	resBody := utils.ReadResBody(artworkDetailsRes)
	err = json.Unmarshal(resBody, &artworkDetailsJsonRes)
	if err != nil {
		utils.LogError(
			err, 
			fmt.Sprintf("Pixiv: Failed to unmarshal artwork details for %s\nJSON: %s", artworkId, string(resBody)),
			false,
		)
		return "", nil, -1
	}
	artworkJsonBody := artworkDetailsJsonRes.Body
	illustratorName := artworkJsonBody.UserName
	artworkName := artworkJsonBody.Title
	artworkPostDir := utils.CreatePostFolder(filepath.Join(downloadPath, api.PixivTitle), illustratorName, artworkId, artworkName)

	artworkType := artworkJsonBody.IllustType
	switch artworkType {
	case illust, manga: // illustration or manga
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/pages", artworkId)
	case ugoira: // ugoira
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/ugoira_meta", artworkId)
	default:
		errorMsg := fmt.Sprintf("Pixiv: Unsupported artwork type %d for artwork ID %s", artworkType, artworkId)
		utils.LogError(nil, errorMsg, false)
		return "", nil, -1
	}

	artworkUrlsRes, err := request.CallRequest("GET", url, 15, cookies, headers, nil, false)
	if err != nil {
		errorMsg := fmt.Sprintf("Pixiv: Failed to get artwork URLs for %s\nURL: %s", artworkId, url)
		utils.LogError(err, errorMsg, false)
		return "", nil, -1
	}
	defer artworkUrlsRes.Body.Close()
	if artworkUrlsRes.StatusCode != 200 {
		errorMsg := fmt.Sprintf(
			"Pixiv: Failed to get details for artwork URLs %v due to %s response\nURL: %s",
			artworkId,
			artworkUrlsRes.Status,
			url,
		)
		utils.LogError(nil, errorMsg, false)
		return "", nil, -1
	}
	return artworkPostDir, utils.LoadJsonFromResponse(artworkUrlsRes), artworkType
}

// Retrieves multiple artwork details based on the given slice of artwork IDs
// and returns a map to use for downloading and a slice of Ugoira structures
func GetMultipleArtworkDetails(artworkIds []string, downloadPath string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	bar := utils.GetProgressBar(
		len(artworkIds), 
		"Getting and processing artwork details...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting and processing %d artwork details!", len(artworkIds))),
	)
	var ugoiraDetails []Ugoira
	var artworkDetails []map[string]string
	lastArtworkId := artworkIds[len(artworkIds)-1]
	for _, artworkId := range artworkIds {
		artworkFolder, jsonRes, artworkType := GetArtworkDetails(artworkId, downloadPath, cookies)
		if artworkFolder == "" || artworkType == -1 {
			continue
		}
		processedJsonRes, ugoiraInfo := ProcessArtworkJson(jsonRes, artworkType, artworkFolder)
		if artworkType == ugoira {
			ugoiraDetails = append(ugoiraDetails, ugoiraInfo)
		} else {
			artworkDetails = append(artworkDetails, processedJsonRes...)
		}

		bar.Add(1)
		if artworkId != lastArtworkId {
			PixivSleep()
		}
	}
	return artworkDetails, ugoiraDetails
}

// Query Pixiv's API for all the illustrator's posts
func GetIllustratorPosts(illustratorId, artworkType string, cookies []http.Cookie) []string {
	artworkType = strings.ToLower(artworkType)
	headers := GetPixivRequestHeaders()
	headers["Referer"] = GetIllustUrl(illustratorId)
	url := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/profile/all", illustratorId)
	res, err := request.CallRequest("GET", url, 5, cookies, headers, nil, false)
	if err != nil {
		utils.LogError(err, "Pixiv: Failed to get illustrator's posts with an ID of " + illustratorId, false)
		return nil
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		errorMsg := fmt.Sprintf(
			"Pixiv: Failed to get illustrator's posts with an ID of %s due to a %s response",
			illustratorId, res.Status,
		)
		utils.LogError(nil, errorMsg, false)
		return nil
	}
	jsonBody := utils.LoadJsonFromResponse(res).(map[string]interface{})["body"]

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
	return artworkIds
}

// Get posts from multiple illustrators and returns a map and a slice for Ugoira structures for downloads
func GetMultipleIllustratorPosts(illustratorIds []string, downloadPath, artworkType string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	bar := utils.GetProgressBar(
		len(illustratorIds), 
		"Getting artwork details from illustrator(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting artwork details from %d illustrator(s)!", len(illustratorIds))),
	)
	var artworkIdsArr []string
	for _, illustratorId := range illustratorIds {
		artworkIds := GetIllustratorPosts(illustratorId, artworkType, cookies)
		artworkIdsArr = append(artworkIdsArr, artworkIds...)
		bar.Add(1)
	}
	artworksArr, ugoiraArr := GetMultipleArtworkDetails(artworkIdsArr, downloadPath, cookies)
	return artworksArr, ugoiraArr
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
func ProcessTagJsonResults(res *http.Response) []string {
	var pixivTagJson PixivTag
	resBody := utils.ReadResBody(res)
	err := json.Unmarshal(resBody, &pixivTagJson)
	if err != nil {
		utils.LogError(err, fmt.Sprintf("Failed to unmarshal json for Pixiv's Tag JSON\nJSON: %s", string(resBody)), false)
	}

	artworksArr := []string{}
	for _, illust := range pixivTagJson.Body.IllustManga.Data {
		artworksArr = append(artworksArr, illust.Id)
	}
	return artworksArr
}

// Query Pixiv's API and search for posts based on the supplied tag name
// which will return a map and a slice of Ugoira structures for downloads
func TagSearch(tagName, downloadPath, sortOrder, searchMode, ratingMode, artworkType string, minPage, maxPage int, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	searchMode = strings.ToLower(searchMode)
	sortOrder = strings.ToLower(sortOrder)
	ratingMode = strings.ToLower(ratingMode)
	artworkType = strings.ToLower(artworkType)

	url := "https://www.pixiv.net/ajax/search/artworks/" + tagName
	params := map[string]string{
		"word": tagName,	  // search term
		"s_mode": searchMode, // search mode: s_tag, s_tag_full, s_tc
		"order": sortOrder,   // sort order: date, popular, popular_male, popular_female 
							  // 			(add "_d" suffix for descending order, e.g. date_d)
		"mode": ratingMode,	  //  r18, safe, or all for both
		"type": artworkType,  // illust_and_ugoira, manga, all
	}

	if minPage > maxPage {
		minPage, maxPage = maxPage, minPage
	}

	var artworkIds []string
	headers := GetPixivRequestHeaders()
	headers["Referer"] = fmt.Sprintf("https://www.pixiv.net/tags/%s/artworks", tagName)
	for page := minPage; page <= maxPage; page++ {
		params["p"] = strconv.Itoa(page) // page number
		res, err := request.CallRequest("GET", url, 5, cookies, headers, params, true)
		if err != nil {
			utils.LogError(err, "Pixiv: Failed to get search results for " + tagName, false)
			continue
		}
		artworkIds = append(artworkIds, ProcessTagJsonResults(res)...)
		if page != maxPage {
			PixivSleep()
		}
	}

	artworkArr, ugoiraArr := GetMultipleArtworkDetails(artworkIds, downloadPath, cookies)
	return artworkArr, ugoiraArr
}