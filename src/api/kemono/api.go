package kemono

import (
	"fmt"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type kemonoChanRes struct {
	urlsToDownload []*request.ToDownload
	gdriveLinks    []*request.ToDownload
	err            error
}

func getPostDetails(post *models.KemonoPostToDl, downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	http3Supported := utils.IsHttp3Supported(utils.KEMONO, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url: fmt.Sprintf(
				"%s/%s/user/%s/post/%s",
				utils.KEMONO_URL,
				post.Service,
				post.CreatorId,
				post.PostId,
			),
			Method:      "GET",
			Cookies:     dlOption.SessionCookies,
			Http2:       !http3Supported,
			Http3:       http3Supported,
			CheckStatus: true,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	var resJson models.KemonoJson
	err = utils.LoadJsonFromResponse(res, &resJson)
	if err != nil {
		return nil, nil, err
	}

	postsToDl, gdriveLinks := processMultipleJson(resJson, downloadPath, dlOption)
	return postsToDl, gdriveLinks, nil
}

func getMultiplePosts(posts []*models.KemonoPostToDl, downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var maxConcurrency int
	if len(posts) > API_MAX_CONCURRENT {
		maxConcurrency = API_MAX_CONCURRENT
	} else {
		maxConcurrency = len(posts)
	}
	wg := sync.WaitGroup{}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *kemonoChanRes, len(posts))
	for _, post := range posts {
		wg.Add(1)
		go func(post *models.KemonoPostToDl) {
			defer func() {
				wg.Done()
				<-queue
			}()

			queue <- struct{}{}
			toDownload, foundGdriveLinks, err := getPostDetails(post, downloadPath, dlOption)
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

	var urlsToDownload, gdriveLinks []*request.ToDownload
	for res := range resChan {
		if res.err != nil {
			utils.LogError(res.err, "", false, utils.ERROR)
			continue
		}
		urlsToDownload = append(urlsToDownload, res.urlsToDownload...)
		gdriveLinks = append(gdriveLinks, res.gdriveLinks...)
	}
	return urlsToDownload, gdriveLinks
}

func getCreatorPosts(creator *models.KemonoCreatorToDl, downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	http3Supported := utils.IsHttp3Supported(utils.KEMONO, true)
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(creator.PageNum)
	if err != nil {
		return nil, nil, err
	}
	minOffset, maxOffset := utils.ConvertPageNumToOffset(minPage, maxPage, utils.KEMONO_PER_PAGE)

	var postsToDl, gdriveLinksToDl []*request.ToDownload
	params := map[string]string{
		"o": fmt.Sprintf("%d", minOffset),
	}
	curOffset := minOffset
	for {
		res, err := request.CallRequest(
			&request.RequestArgs{
				Url: fmt.Sprintf(
					"%s/%s/user/%s",
					utils.KEMONO_URL,
					creator.Service,
					creator.CreatorId,
				),
				Method:      "GET",
				Cookies:     dlOption.SessionCookies,
				Params:      params,
				Http2:       !http3Supported,
				Http3:       http3Supported,
				CheckStatus: true,
			},
		)
		if err != nil {
			return nil, nil, err
		}

		var resJson models.KemonoJson
		err = utils.LoadJsonFromResponse(res, &resJson)
		if err != nil {
			return nil, nil, err
		}

		if len(resJson) == 0 {
			break
		}

		posts, gdriveLinks := processMultipleJson(resJson, downloadPath, dlOption)
		postsToDl = append(postsToDl, posts...)
		gdriveLinksToDl = append(gdriveLinksToDl, gdriveLinks...)

		if (hasMax && curOffset >= maxOffset) {
			break
		}
		curOffset += 25
	}
	return postsToDl, gdriveLinksToDl, nil
}

func getMultipleCreators(creators []*models.KemonoCreatorToDl, downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var errSlice []error
	var urlsToDownload, gdriveLinks []*request.ToDownload
	for _, creator := range creators {
		postsToDl, gdriveLinksToDl, err := getCreatorPosts(creator, downloadPath, dlOption)
		if err != nil {
			errSlice = append(errSlice, err)
		}
		urlsToDownload = append(urlsToDownload, postsToDl...)
		gdriveLinks = append(gdriveLinks, gdriveLinksToDl...)
	}

	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
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

func getFavourites(downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	http3Supported := utils.IsHttp3Supported(utils.KEMONO, true)
	params := map[string]string{
		"type": "artist",
	}
	reqArgs := &request.RequestArgs{
		Url:         fmt.Sprintf("%s/v1/account/favorites", utils.KEMONO_API_URL),
		Method:      "GET",
		Cookies:     dlOption.SessionCookies,
		Params:      params,
		Http2:       !http3Supported,
		Http3:       http3Supported,
		CheckStatus: true,
	}
	res, err := request.CallRequest(reqArgs)
	if err != nil {
		return nil, nil, err
	}

	var creatorResJson models.KemonoFavCreatorJson
	err = utils.LoadJsonFromResponse(res, &creatorResJson)
	if err != nil {
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
	err = utils.LoadJsonFromResponse(res, &postResJson)
	if err != nil {
		return nil, nil, err
	}
	urlsToDownload, gdriveLinks := processMultipleJson(postResJson, downloadPath, dlOption)

	creatorsPost, creatorsGdrive := getMultipleCreators(artistToDl, downloadPath, dlOption)
	urlsToDownload = append(urlsToDownload, creatorsPost...)
	gdriveLinks = append(gdriveLinks, creatorsGdrive...)

	return urlsToDownload, gdriveLinks, nil
}
