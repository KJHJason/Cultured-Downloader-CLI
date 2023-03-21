package pixivweb

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/common"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func getArtworkDetailsLogic(artworkId string, reqArgs *request.RequestArgs) (*models.ArtworkDetails, error) {
	artworkDetailsRes, err := request.CallRequest(reqArgs)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork details for ID %v from %s",
			utils.CONNECTION_ERROR,
			artworkId,
			reqArgs.Url,
		)
		return nil, err
	}

	if artworkDetailsRes.StatusCode != 200 {
		artworkDetailsRes.Body.Close()
		err = fmt.Errorf(
			"pixiv error %d: failed to get details for artwork ID %s due to %s response from %s",
			utils.RESPONSE_ERROR,
			artworkId,
			artworkDetailsRes.Status,
			reqArgs.Url,
		)
		return nil, err
	}

	var artworkDetailsJsonRes models.ArtworkDetails
	err = utils.LoadJsonFromResponse(artworkDetailsRes, &artworkDetailsJsonRes)
	if err != nil {
		err = fmt.Errorf(
			"%v\ndetails: failed to read response body for Pixiv artwork ID %s",
			err,
			artworkId,
		)
		return nil, err
	}
	return &artworkDetailsJsonRes, nil
}

func getArtworkUrlsToDlLogic(artworkType int64, artworkId string, reqArgs *request.RequestArgs) (*http.Response, error) {
	var url string
	switch artworkType {
	case ILLUST, MANGA: // illustration or manga
		url = fmt.Sprintf("%s/illust/%s/pages", utils.PIXIV_API_URL, artworkId)
	case UGOIRA: // ugoira
		url = fmt.Sprintf("%s/illust/%s/ugoira_meta", utils.PIXIV_API_URL, artworkId)
	default:
		return nil, fmt.Errorf(
			"pixiv error %d: unsupported artwork type %d for artwork ID %s",
			utils.JSON_ERROR,
			artworkType,
			artworkId,
		)
	}

	reqArgs.Url = url
	artworkUrlsRes, err := request.CallRequest(reqArgs)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to get artwork URLs for ID %s from %s due to %v",
			utils.CONNECTION_ERROR,
			artworkId,
			url,
			err,
		)
		return nil, err
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
		return nil, err
	}
	return artworkUrlsRes, nil
}

// Retrieves details of an artwork ID and returns
// the folder path to download the artwork to, the JSON response, and the artwork type
func getArtworkDetails(artworkId, downloadPath string, dlOptions *PixivWebDlOptions) ([]*request.ToDownload, *models.Ugoira, error) {
	if artworkId == "" {
		return nil, nil, nil
	}

	url := fmt.Sprintf("%s/illust/%s", utils.PIXIV_API_URL, artworkId)
	headers := pixivcommon.GetPixivRequestHeaders()
	headers["Referer"] = pixivcommon.GetUserUrl(artworkId)

	useHttp3 := utils.IsHttp3Supported(utils.PIXIV, true)
	reqArgs := &request.RequestArgs{
		Url:       url,
		Method:    "GET",
		Cookies:   dlOptions.SessionCookies,
		Headers:   headers,
		UserAgent: dlOptions.Configs.UserAgent,
		Http2:     !useHttp3,
		Http3:     useHttp3,
	}
	artworkDetailsJsonRes, err := getArtworkDetailsLogic(artworkId, reqArgs)
	if err != nil {
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
	artworkUrlsRes, err := getArtworkUrlsToDlLogic(artworkType, artworkId, reqArgs)
	if err != nil {
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
func GetMultipleArtworkDetails(artworkIds []string, downloadPath string, dlOptions *PixivWebDlOptions) ([]*request.ToDownload, []*models.Ugoira) {
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
			dlOptions,
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
func getIllustratorPosts(illustratorId, pageNum string, dlOptions *PixivWebDlOptions) ([]string, error) {
	headers := pixivcommon.GetPixivRequestHeaders()
	headers["Referer"] = pixivcommon.GetIllustUrl(illustratorId)
	url := fmt.Sprintf("%s/user/%s/profile/all", utils.PIXIV_API_URL, illustratorId)

	useHttp3 := utils.IsHttp3Supported(utils.PIXIV, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:       url,
			Method:    "GET",
			Cookies:   dlOptions.SessionCookies,
			Headers:   headers,
			UserAgent: dlOptions.Configs.UserAgent,
			Http2:     !useHttp3,
			Http3:     useHttp3,
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
	artworkIds, err := processIllustratorPostJson(&jsonBody, pageNum, dlOptions)
	return artworkIds, err
}

// Get posts from multiple illustrators and returns a slice of artwork IDs
func GetMultipleIllustratorPosts(illustratorIds, pageNums []string, downloadPath string, dlOptions *PixivWebDlOptions) []string {
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
			dlOptions,
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

type pageNumArgs struct {
	minPage int
	maxPage int
	hasMax  bool
}

func tagSearchLogic(tagName string, reqArgs *request.RequestArgs, pageNumArgs *pageNumArgs) ([]string, []error) {
	var errSlice []error
	var artworkIds []string
	page := 0
	for {
		page++
		if page < pageNumArgs.minPage {
			continue
		}
		if pageNumArgs.hasMax && page > pageNumArgs.maxPage {
			break
		}

		reqArgs.Params["p"] = strconv.Itoa(page) // page number
		res, err := request.CallRequest(reqArgs)
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
		if page != pageNumArgs.maxPage {
			pixivSleep()
		}
	}
	return artworkIds, errSlice
}

// Query Pixiv's API and search for posts based on the supplied tag name
// which will return a map and a slice of Ugoira structures for downloads
func TagSearch(tagName, downloadPath, pageNum string, dlOptions *PixivWebDlOptions) ([]*request.ToDownload, []*models.Ugoira, bool) {
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

	useHttp3 := utils.IsHttp3Supported(utils.PIXIV, true)
	headers := pixivcommon.GetPixivRequestHeaders()
	headers["Referer"] = fmt.Sprintf("%s/tags/%s/artworks", utils.PIXIV_URL, tagName)
	artworkIds, errSlice := tagSearchLogic(
		tagName,
		&request.RequestArgs{
			Url:         url,
			Method:      "GET",
			Cookies:     dlOptions.SessionCookies,
			Headers:     headers,
			Params:      params,
			CheckStatus: true,
			UserAgent:   dlOptions.Configs.UserAgent,
			Http2:       !useHttp3,
			Http3:       useHttp3,
		},
		&pageNumArgs{
			minPage: minPage,
			maxPage: maxPage,
			hasMax:  hasMax,
		},
	)

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}

	artworkSlice, ugoiraSlice := GetMultipleArtworkDetails(
		artworkIds,
		downloadPath,
		dlOptions,
	)
	return artworkSlice, ugoiraSlice, hasErr
}
