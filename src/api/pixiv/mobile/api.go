package pixivmobile

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/common"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/ugoira"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type offsetArgs struct {
	minOffset int
	maxOffset int
	hasMax    bool
}

// Returns the Ugoira structure with the necessary information to download the ugoira
//
// Will return an error which has been logged if unexpected error occurs like connection error, json marshal error, etc.
func (pixiv *PixivMobile) getUgoiraMetadata(illustId, dlFilePath string) (*models.Ugoira, error) {
	ugoiraUrl := pixiv.baseUrl + "/v1/ugoira/metadata"
	params := map[string]string{"illust_id": illustId}
	additionalHeaders := pixiv.getHeaders(
		map[string]string{"Referer": pixiv.baseUrl},
	)

	res, err := pixiv.SendRequest(
		&request.RequestArgs{
			Url:         ugoiraUrl,
			CheckStatus: true,
			Headers:     additionalHeaders,
			Params:      params,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"pixiv movile error %d: Failed to get ugoira metadata for %s",
			utils.CONNECTION_ERROR,
			illustId,
		)
	}

	var ugoiraJson models.UgoiraJson
	if err := utils.LoadJsonFromResponse(res, &ugoiraJson); err != nil {
		return nil, err
	}

	ugoiraMetadata := ugoiraJson.Metadata
	ugoiraDlUrl := ugoiraMetadata.ZipUrls.Medium
	ugoiraDlUrl = strings.Replace(ugoiraDlUrl, "600x600", "1920x1080", 1)

	// map the files to their delay
	frameInfoMap := ugoira.MapDelaysToFilename(ugoiraMetadata.Frames)
	return &models.Ugoira{
		Url:      ugoiraDlUrl,
		Frames:   frameInfoMap,
		FilePath: dlFilePath,
	}, nil
}

// Query Pixiv's API (mobile) to get the JSON of an artwork ID
func (pixiv *PixivMobile) getArtworkDetails(artworkId, downloadPath string) ([]*request.ToDownload, *models.Ugoira, error) {
	artworkUrl := pixiv.baseUrl + "/v1/illust/detail"
	params := map[string]string{"illust_id": artworkId}

	res, err := pixiv.SendRequest(
		&request.RequestArgs{
			Url:         artworkUrl,
			Params:      params,
			CheckStatus: true,
		},
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

	var artworkJson models.PixivMobileArtworkJson
	if err := utils.LoadJsonFromResponse(res, &artworkJson); err != nil {
		return nil, nil, err
	}

	artworkDetails, ugoiraToDl, err := pixiv.processArtworkJson(
		artworkJson.Illust,
		downloadPath,
	)
	return artworkDetails, ugoiraToDl, err
}

func (pixiv *PixivMobile) GetMultipleArtworkDetails(artworkIds []string, downloadPath string) ([]*request.ToDownload, []*models.Ugoira) {
	var artworksToDownload []*request.ToDownload
	var ugoiraSlice []*models.Ugoira
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
		artworkDetails, ugoiraInfo, err := pixiv.getArtworkDetails(artworkId, downloadPath)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		}

		artworksToDownload = append(artworksToDownload, artworkDetails...)
		ugoiraSlice = append(ugoiraSlice, ugoiraInfo)
		if idx != lastIdx {
			pixiv.Sleep()
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)

	return artworksToDownload, ugoiraSlice
}

func (pixiv *PixivMobile) getIllustratorPostMainLogic(params map[string]string, userId, downloadPath string, offsetArg *offsetArgs) ([]*request.ToDownload, []*models.Ugoira, []error) {
	var errSlice []error
	var ugoiraSlice []*models.Ugoira
	var artworksToDownload []*request.ToDownload
	nextUrl := pixiv.baseUrl + "/v1/user/illusts"

	curOffset := offsetArg.minOffset
	for nextUrl != "" {
		res, err := pixiv.SendRequest(
			&request.RequestArgs{
				Url:         nextUrl,
				Params:      params,
				CheckStatus: true,
			},
		)
		if err != nil {
			err = fmt.Errorf(
				"pixiv mobile error %d: failed to get illustrator posts for %s, more info => %v",
				utils.CONNECTION_ERROR,
				userId,
				err,
			)
			return nil, nil, []error{err}
		}

		var resJson models.PixivMobileArtworksJson
		if err := utils.LoadJsonFromResponse(res, &resJson); err != nil {
			return nil, nil, []error{err}
		}

		artworks, ugoira, errS := pixiv.processMultipleArtworkJson(&resJson, downloadPath)
		if len(errS) > 0 {
			errSlice = append(errSlice, errS...)
		}
		artworksToDownload = append(artworksToDownload, artworks...)
		ugoiraSlice = append(ugoiraSlice, ugoira...)

		curOffset += 30
		params["offset"] = strconv.Itoa(curOffset)
		jsonNextUrl := resJson.NextUrl
		if jsonNextUrl == nil || (offsetArg.hasMax && curOffset >= offsetArg.maxOffset) {
			nextUrl = ""
		} else {
			nextUrl = *jsonNextUrl
			pixiv.Sleep()
		}
	}
	return artworksToDownload, ugoiraSlice, errSlice
}

// Query Pixiv's API (mobile) to get all the posts JSON(s) of a user ID
func (pixiv *PixivMobile) getIllustratorPosts(userId, pageNum, downloadPath, artworkType string) ([]*request.ToDownload, []*models.Ugoira, []error) {
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, nil, []error{err}
	}
	minOffset, maxOffset := pixivcommon.ConvertPageNumToOffset(minPage, maxPage, utils.PIXIV_PER_PAGE, false)

	params := map[string]string{
		"user_id": userId,
		"filter":  "for_ios",
		"offset":  strconv.Itoa(minOffset),
		"type":    artworkType,
	}
	if artworkType == "all" {
		params["type"] = "illust"
	}

	offsetArgs := &offsetArgs{
		minOffset: minOffset,
		maxOffset: maxOffset,
		hasMax:    hasMax,
	}
	artworksToDl, ugoiraSlice, errSlice := pixiv.getIllustratorPostMainLogic(
		params,
		userId,
		downloadPath,
		offsetArgs,
	)

	if params["type"] == "illust" && artworkType == "all" {
		// if the user is downloading both
		// illust and manga, loop again to get the manga
		params["type"] = "manga"
		artworksToDl2, ugoiraSlice2, errSlice2 := pixiv.getIllustratorPostMainLogic(
			params,
			userId,
			downloadPath,
			offsetArgs,
		)
		artworksToDl = append(artworksToDl, artworksToDl2...)
		ugoiraSlice = append(ugoiraSlice, ugoiraSlice2...)
		errSlice = append(errSlice, errSlice2...)
	}
	return artworksToDl, ugoiraSlice, errSlice
}

func (pixiv *PixivMobile) GetMultipleIllustratorPosts(userIds, pageNums []string, downloadPath, artworkType string) ([]*request.ToDownload, []*models.Ugoira) {
	userIdsLen := len(userIds)
	lastIdx := userIdsLen - 1

	var errSlice []error
	var ugoiraSlice []*models.Ugoira
	var artworksToDownload []*request.ToDownload
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
		artworkDetails, ugoiraInfo, err := pixiv.getIllustratorPosts(
			userId,
			pageNums[idx],
			downloadPath,
			artworkType,
		)
		if err != nil {
			errSlice = append(errSlice, err...)
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
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)

	return artworksToDownload, ugoiraSlice
}

func (pixiv *PixivMobile) tagSearchLogic(tagName, downloadPath string, dlOptions *PixivMobileDlOptions, offsetArg *offsetArgs) ([]*request.ToDownload, []*models.Ugoira, []error) {
	var errSlice []error
	var ugoiraSlice []*models.Ugoira
	var artworksToDownload []*request.ToDownload
	params := map[string]string{
		"word":          tagName,
		"search_target": dlOptions.SearchMode,
		"sort":          dlOptions.SortOrder,
		"filter":        "for_ios",
		"offset":        strconv.Itoa(offsetArg.minOffset),
	}
	curOffset := offsetArg.minOffset
	nextUrl := pixiv.baseUrl + "/v1/search/illust"
	for nextUrl != "" {
		res, err := pixiv.SendRequest(
			&request.RequestArgs{
				Url:         nextUrl,
				Params:      params,
				CheckStatus: true,
			},
		)
		if err != nil {
			err = fmt.Errorf(
				"pixiv mobile error %d: failed to search for %q, more info => %v",
				utils.CONNECTION_ERROR,
				tagName,
				err,
			)
			return nil, nil, []error{err} 
		}

		var resJson models.PixivMobileArtworksJson
		if err := utils.LoadJsonFromResponse(res, &resJson); err != nil {
			errSlice = append(errSlice, err)
			continue
		}

		artworks, ugoira, errS := pixiv.processMultipleArtworkJson(&resJson, downloadPath)
		errSlice = append(errSlice, errS...)
		artworksToDownload = append(artworksToDownload, artworks...)
		ugoiraSlice = append(ugoiraSlice, ugoira...)

		curOffset += 30
		params["offset"] = strconv.Itoa(curOffset)
		jsonNextUrl := resJson.NextUrl
		if jsonNextUrl == nil || (offsetArg.hasMax && curOffset >= offsetArg.maxOffset) {
			nextUrl = ""
		} else {
			nextUrl = *jsonNextUrl
			pixiv.Sleep()
		}
	}
	return artworksToDownload, ugoiraSlice, errSlice
}

// Query Pixiv's API (mobile) to get the JSON of a search query
func (pixiv *PixivMobile) TagSearch(tagName, downloadPath, pageNum string, dlOptions *PixivMobileDlOptions) ([]*request.ToDownload, []*models.Ugoira, bool) {
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		utils.LogError(
			err,
			"",
			false,
			utils.ERROR,
		)
		return nil, nil, true
	}
	minOffset, maxOffset := pixivcommon.ConvertPageNumToOffset(minPage, maxPage, utils.PIXIV_PER_PAGE, false)

	artworksToDl, ugoiraSlice, errSlice := pixiv.tagSearchLogic(
		tagName,
		downloadPath,
		dlOptions,
		&offsetArgs{
			minOffset: minOffset,
			maxOffset: maxOffset,
			hasMax:    hasMax,
		},
	)
	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	return artworksToDl, ugoiraSlice, len(errSlice) > 0
}
