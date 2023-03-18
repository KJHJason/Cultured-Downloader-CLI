package pixivweb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/common"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Retrieves details of an artwork ID and returns
// the folder path to download the artwork to, the JSON response, and the artwork type
func getArtworkDetails(artworkId, downloadPath string, config *configs.Config, cookies []*http.Cookie) ([]*request.ToDownload, *models.Ugoira, error) {
	if artworkId == "" {
		return nil, nil, nil
	}

	headers := pixivcommon.GetPixivRequestHeaders()
	headers["Referer"] = pixivcommon.GetUserUrl(artworkId)
	url := fmt.Sprintf("%s/illust/%s", utils.PIXIV_API_URL, artworkId)
	artworkDetailsRes, err := request.CallRequest(
		&request.RequestArgs{
			Url:       url,
			Method:    "GET",
			Cookies:   cookies,
			Headers:   headers,
			UserAgent: config.UserAgent,
		},
	)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork details for ID %v from %s",
			utils.CONNECTION_ERROR,
			artworkId,
			url,
		)
		return nil, nil, err
	}

	if artworkDetailsRes == nil {
		return nil, nil, nil
	}

	if artworkDetailsRes.StatusCode != 200 {
		artworkDetailsRes.Body.Close()
		err = fmt.Errorf(
			"pixiv error %d: failed to get details for artwork ID %s due to %s response from %s",
			utils.RESPONSE_ERROR,
			artworkId,
			artworkDetailsRes.Status,
			url,
		)
		return nil, nil, err
	}
	var artworkDetailsJsonRes models.ArtworkDetails
	resBody, err := utils.ReadResBody(artworkDetailsRes)
	if err != nil {
		err = fmt.Errorf(
			"%v\ndetails: failed to read response body for Pixiv artwork ID %s",
			err,
			artworkId,
		)
		return nil, nil, err
	}

	err = json.Unmarshal(resBody, &artworkDetailsJsonRes)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to unmarshal artwork details for ID %s\nJSON response: %s",
			utils.JSON_ERROR,
			artworkId,
			string(resBody),
		)
		return nil, nil, err
	}
	artworkJsonBody := artworkDetailsJsonRes.Body
	illustratorName := artworkJsonBody.UserName
	artworkName := artworkJsonBody.Title
	artworkPostDir := utils.GetPostFolder(
		filepath.Join(downloadPath, utils.PIXIV_TITLE),
		illustratorName,
		artworkId,
		artworkName,
	)

	artworkType := artworkJsonBody.IllustType
	switch artworkType {
	case ILLUST, MANGA: // illustration or manga
		url = fmt.Sprintf("%s/illust/%s/pages", utils.PIXIV_API_URL, artworkId)
	case UGOIRA: // ugoira
		url = fmt.Sprintf("%s/illust/%s/ugoira_meta", utils.PIXIV_API_URL, artworkId)
	default:
		err = fmt.Errorf(
			"pixiv error %d: unsupported artwork type %d for artwork ID %s",
			utils.JSON_ERROR,
			artworkType,
			artworkId,
		)
		return nil, nil, err
	}

	artworkUrlsRes, err := request.CallRequest(
		&request.RequestArgs{
			Url:       url,
			Method:    "GET",
			Cookies:   cookies,
			Headers:   headers,
			UserAgent: config.UserAgent,
		},
	)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork URLs for ID %s from %s due to %v",
			utils.CONNECTION_ERROR,
			artworkId,
			url,
			err,
		)
		return nil, nil, err
	}

	if artworkUrlsRes.StatusCode != 200 {
		artworkUrlsRes.Body.Close()
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork URLs for ID %s due to %s response from %s",
			utils.RESPONSE_ERROR,
			artworkId,
			artworkUrlsRes.Status,
			url,
		)
		return nil, nil, err
	}

	urlsToDl, ugoiraInfo, err := processArtworkJson(
		artworkUrlsRes,
		artworkType,
		artworkPostDir,
	)
	if err != nil {
		return nil, nil, err
	}
	return urlsToDl, ugoiraInfo, nil
}

// Retrieves multiple artwork details based on the given slice of artwork IDs
// and returns a map to use for downloading and a slice of Ugoira structures
func GetMultipleArtworkDetails(artworkIds []string, downloadPath string, config *configs.Config, cookies []*http.Cookie) ([]*request.ToDownload, []*models.Ugoira) {
	var errSlice []error
	var ugoiraDetails []*models.Ugoira
	var artworkDetails []*request.ToDownload
	artworkIdsLen := len(artworkIds)
	lastArtworkId := artworkIds[artworkIdsLen-1]

	baseMsg := "Getting and processing artwork details from Pixiv [%d/" + fmt.Sprintf("%d]...", artworkIdsLen)
	progress := spinner.New(
		spinner.JSON_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting and processing %d artwork details from Pixiv!",
			artworkIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting and processing %d artwork details from Pixiv!\nPlease refer to the logs for more details.",
			artworkIdsLen,
		),
		artworkIdsLen,
	)
	progress.Start()
	for _, artworkId := range artworkIds {
		artworksToDl, ugoiraInfo, err := getArtworkDetails(
			artworkId,
			downloadPath,
			config,
			cookies,
		)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		}

		if ugoiraInfo != nil {
			ugoiraDetails = append(ugoiraDetails, ugoiraInfo)
		} else {
			artworkDetails = append(artworkDetails, artworksToDl...)
		}

		progress.MsgIncrement(baseMsg)
		if artworkId != lastArtworkId {
			pixivSleep()
		}
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)

	return artworkDetails, ugoiraDetails
}

// Query Pixiv's API for all the illustrator's posts
func getIllustratorPosts(illustratorId, pageNum string, config *configs.Config, pixivDlOptions *PixivWebDlOptions) ([]string, error) {
	headers := pixivcommon.GetPixivRequestHeaders()
	headers["Referer"] = pixivcommon.GetIllustUrl(illustratorId)
	url := fmt.Sprintf("%s/user/%s/profile/all", utils.PIXIV_API_URL, illustratorId)

	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:       url,
			Method:    "GET",
			Cookies:   pixivDlOptions.SessionCookies,
			Headers:   headers,
			UserAgent: config.UserAgent,
		},
	)
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

	var jsonBody models.PixivWebIllustratorJson
	err = utils.LoadJsonFromResponse(res, &jsonBody)
	if err != nil {
		return nil, err
	}

	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, err
	}
	minOffset, maxOffset := pixivcommon.ConvertPageNumToOffset(minPage, maxPage, utils.PIXIV_PER_PAGE, false)

	var artworkIds []string
	if pixivDlOptions.ArtworkType == "all" || pixivDlOptions.ArtworkType == "illust_and_ugoira" {
		illusts := jsonBody.Body.Illusts
		switch t := illusts.(type) {
		case map[string]interface{}:
			curOffset := 0
			for illustId := range t {
				curOffset++
				if curOffset < minOffset {
					continue
				}
				if hasMax && curOffset > maxOffset {
					break
				}

				artworkIds = append(artworkIds, illustId)
			}
		default: // where there are no posts or has an unknown type
			break
		}
	}

	if pixivDlOptions.ArtworkType == "all" || pixivDlOptions.ArtworkType == "manga" {
		manga := jsonBody.Body.Manga
		switch t := manga.(type) {
		case map[string]interface{}:
			curOffset := 0
			for mangaId := range t {
				curOffset++
				if curOffset < minOffset {
					continue
				}
				if hasMax && curOffset > maxOffset {
					break
				}

				artworkIds = append(artworkIds, mangaId)
			}
		default: // where there are no posts or has an unknown type
			break
		}
	}
	return artworkIds, nil
}

// Get posts from multiple illustrators and returns a slice of artwork IDs
func GetMultipleIllustratorPosts(illustratorIds, pageNums []string, downloadPath string, config *configs.Config, pixivDlOptions *PixivWebDlOptions) []string {
	var errSlice []error
	var artworkIdsSlice []string
	illustratorIdsLen := len(illustratorIds)
	lastIllustratorIdx := illustratorIdsLen - 1

	baseMsg := "Getting artwork details from illustrator(s) on Pixiv [%d/" + fmt.Sprintf("%d]...", illustratorIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting artwork details from %d illustrator(s) on Pixiv!",
			illustratorIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting artwork details from %d illustrator(s) on Pixiv!\nPlease refer to the logs for more details.",
			illustratorIdsLen,
		),
		illustratorIdsLen,
	)
	progress.Start()
	for idx, illustratorId := range illustratorIds {
		artworkIds, err := getIllustratorPosts(
			illustratorId,
			pageNums[idx],
			config,
			pixivDlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			artworkIdsSlice = append(artworkIdsSlice, artworkIds...)
		}

		if idx != lastIllustratorIdx {
			pixivSleep()
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)

	return artworkIdsSlice
}

// Query Pixiv's API and search for posts based on the supplied tag name
// which will return a map and a slice of Ugoira structures for downloads
func TagSearch(tagName, downloadPath, pageNum string, config *configs.Config, dlOptions *PixivWebDlOptions) ([]*request.ToDownload, []*models.Ugoira, bool) {
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		utils.LogError(err, "", false, utils.ERROR)
		return nil, nil, true
	}

	url := fmt.Sprintf("%s/search/artworks/%s", utils.PIXIV_API_URL, tagName)
	params := map[string]string{
		// search term
		"word": tagName,

		// search mode: s_tag, s_tag_full, s_tc
		"s_mode": dlOptions.SearchMode,

		// sort order: date, popular, popular_male, popular_female
		// (add "_d" suffix for descending order, e.g. date_d)
		"order": dlOptions.SortOrder,

		//  r18, safe, or all for both
		"mode": dlOptions.RatingMode,

		// illust_and_ugoira, manga, all
		"type": dlOptions.ArtworkType,
	}

	var errSlice []error
	var artworkIds []string
	headers := pixivcommon.GetPixivRequestHeaders()
	headers["Referer"] = fmt.Sprintf(
		"%s/tags/%s/artworks",
		utils.PIXIV_URL,
		tagName,
	)
	page := 0
	for {
		page++
		if page < minPage {
			continue
		}
		if hasMax && page > maxPage {
			break
		}

		params["p"] = strconv.Itoa(page) // page number
		res, err := request.CallRequest(
			&request.RequestArgs{
				Url:         url,
				Method:      "GET",
				Cookies:     dlOptions.SessionCookies,
				Headers:     headers,
				Params:      params,
				CheckStatus: true,
				UserAgent:   config.UserAgent,
			},
		)
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

		tagArtworkIds, err := processTagJsonResults(res)
		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}

		if len(tagArtworkIds) == 0 {
			break
		}

		artworkIds = append(artworkIds, tagArtworkIds...)
		if page != maxPage {
			pixivSleep()
		}
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}

	artworkSlice, ugoiraSlice := GetMultipleArtworkDetails(
		artworkIds,
		downloadPath,
		config,
		dlOptions.SessionCookies,
	)
	return artworkSlice, ugoiraSlice, hasErr
}
