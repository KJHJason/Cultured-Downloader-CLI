package kemono

import (
	"fmt"
	"sync"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type kemonoChanRes struct {
	urlsToDownload []*request.ToDownload
	gdriveLinks    []*request.ToDownload
	err            error
}

func getKemonoPartyHeaders() map[string]string {
	return map[string]string{
		"Host": utils.KEMONO_URL,
	}
}

func getPostDetails(post *models.KemonoPostToDl, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	useHttp3 := utils.IsHttp3Supported(utils.KEMONO, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url: fmt.Sprintf(
				"%s/%s/user/%s/post/%s",
				utils.KEMONO_API_URL,
				post.Service,
				post.CreatorId,
				post.PostId,
			),
			Method:      "GET",
			Headers:     getKemonoPartyHeaders(),
			UserAgent:   dlOptions.Configs.UserAgent,
			Cookies:     dlOptions.SessionCookies,
			Http2:       !useHttp3,
			Http3:       useHttp3,
			CheckStatus: true,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	var resJson models.KemonoJson
	if err := utils.LoadJsonFromResponse(res, &resJson); err != nil {
		return nil, nil, err
	}

	postsToDl, gdriveLinks := processMultipleJson(resJson, downloadPath, dlOptions)
	return postsToDl, gdriveLinks, nil
}

func getMultiplePosts(posts []*models.KemonoPostToDl, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var maxConcurrency int
	postLen := len(posts)
	if postLen > API_MAX_CONCURRENT {
		maxConcurrency = API_MAX_CONCURRENT
	} else {
		maxConcurrency = postLen
	}
	wg := sync.WaitGroup{}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *kemonoChanRes, postLen)

	baseMsg := "Getting post details from Kemono Party [%d/" + fmt.Sprintf("%d]...", postLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting %d post details from Kemono Party!",
			postLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting %d post details from Kemono Party.\nPlease refer to the logs for more details.",
			postLen,
		),
		postLen,
	)
	progress.Start()
	for _, post := range posts {
		wg.Add(1)
		go func(post *models.KemonoPostToDl) {
			defer func() {
				progress.MsgIncrement(baseMsg)
				wg.Done()
				<-queue
			}()

			queue <- struct{}{}
			toDownload, foundGdriveLinks, err := getPostDetails(post, downloadPath, dlOptions)
			if err != nil {
				resChan <- &kemonoChanRes{
					err: err,
				}
				return
			}
			resChan <- &kemonoChanRes{
				urlsToDownload: toDownload,
				gdriveLinks:    foundGdriveLinks,
			}
		}(post)
	}
	wg.Wait()
	close(queue)
	close(resChan)

	hasError := false
	var urlsToDownload, gdriveLinks []*request.ToDownload
	for res := range resChan {
		if res.err != nil {
			if !hasError {
				hasError = true
			}
			utils.LogError(res.err, "", false, utils.ERROR)
			continue
		}
		urlsToDownload = append(urlsToDownload, res.urlsToDownload...)
		gdriveLinks = append(gdriveLinks, res.gdriveLinks...)
	}
	progress.Stop(hasError)
	return urlsToDownload, gdriveLinks
}

func getCreatorPosts(creator *models.KemonoCreatorToDl, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	useHttp3 := utils.IsHttp3Supported(utils.KEMONO, true)
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(creator.PageNum)
	if err != nil {
		return nil, nil, err
	}
	minOffset, maxOffset := utils.ConvertPageNumToOffset(minPage, maxPage, utils.KEMONO_PER_PAGE)

	var postsToDl, gdriveLinksToDl []*request.ToDownload
	params := make(map[string]string)
	curOffset := minOffset
	for {
		params["o"] = strconv.Itoa(curOffset)
		res, err := request.CallRequest(
			&request.RequestArgs{
				Url: fmt.Sprintf(
					"%s/%s/user/%s",
					utils.KEMONO_API_URL,
					creator.Service,
					creator.CreatorId,
				),
				Method:      "GET",
				UserAgent:   dlOptions.Configs.UserAgent,
				Headers:     getKemonoPartyHeaders(),
				Cookies:     dlOptions.SessionCookies,
				Params:      params,
				Http2:       !useHttp3,
				Http3:       useHttp3,
				CheckStatus: true,
			},
		)
		if err != nil {
			return nil, nil, err
		}

		var resJson models.KemonoJson
		if err := utils.LoadJsonFromResponse(res, &resJson); err != nil {
			return nil, nil, err
		}

		if len(resJson) == 0 {
			break
		}

		posts, gdriveLinks := processMultipleJson(resJson, downloadPath, dlOptions)
		postsToDl = append(postsToDl, posts...)
		gdriveLinksToDl = append(gdriveLinksToDl, gdriveLinks...)

		if (hasMax && curOffset >= maxOffset) {
			break
		}
		curOffset += 25
	}
	return postsToDl, gdriveLinksToDl, nil
}

func getMultipleCreators(creators []*models.KemonoCreatorToDl, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var errSlice []error
	var urlsToDownload, gdriveLinks []*request.ToDownload
	creatorLen := len(creators)
	baseMsg := "Getting creator's posts from Kemono Party [%d/" + fmt.Sprintf("%d]...", creatorLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting %d creator's posts from Kemono Party!",
			creatorLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting %d creator's posts from Kemono Party.\nPlease refer to the logs for more details.",
			creatorLen,
		),
		creatorLen,
	)
	progress.Start()
	for _, creator := range creators {
		postsToDl, gdriveLinksToDl, err := getCreatorPosts(creator, downloadPath, dlOptions)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		}
		urlsToDownload = append(urlsToDownload, postsToDl...)
		gdriveLinks = append(gdriveLinks, gdriveLinksToDl...)
		progress.MsgIncrement(baseMsg)
	}

	hasError := false
	if len(errSlice) > 0 {
		hasError = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasError)
	return urlsToDownload, gdriveLinks
}

func processFavCreator(resJson models.KemonoFavCreatorJson) []*models.KemonoCreatorToDl {
	var creators []*models.KemonoCreatorToDl
	for _, creator := range resJson {
		creators = append(creators, &models.KemonoCreatorToDl{
			CreatorId: creator.Id,
			Service:   creator.Service,
			PageNum:   "", // download all pages
		})
	}
	return creators
}

func getFavourites(downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	useHttp3 := utils.IsHttp3Supported(utils.KEMONO, true)
	params := map[string]string{
		"type": "artist",
	}
	reqArgs := &request.RequestArgs{
		Url:         fmt.Sprintf("%s/v1/account/favorites", utils.KEMONO_API_URL),
		Method:      "GET",
		Cookies:     dlOptions.SessionCookies,
		Params:      params,
		Headers:     getKemonoPartyHeaders(),
		UserAgent:   dlOptions.Configs.UserAgent,
		Http2:       !useHttp3,
		Http3:       useHttp3,
		CheckStatus: true,
	}
	res, err := request.CallRequest(reqArgs)
	if err != nil {
		return nil, nil, err
	}

	var creatorResJson models.KemonoFavCreatorJson
	if err := utils.LoadJsonFromResponse(res, &creatorResJson); err != nil {
		return nil, nil, err
	}
	artistToDl := processFavCreator(creatorResJson)

	reqArgs.Params = map[string]string{
		"type": "post",
	}
	res, err = request.CallRequest(reqArgs)
	if err != nil {
		return nil, nil, err
	}

	var postResJson models.KemonoJson
	if err := utils.LoadJsonFromResponse(res, &postResJson); err != nil {
		return nil, nil, err
	}
	urlsToDownload, gdriveLinks := processMultipleJson(postResJson, downloadPath, dlOptions)

	creatorsPost, creatorsGdrive := getMultipleCreators(artistToDl, downloadPath, dlOptions)
	urlsToDownload = append(urlsToDownload, creatorsPost...)
	gdriveLinks = append(gdriveLinks, creatorsGdrive...)

	return urlsToDownload, gdriveLinks, nil
}
