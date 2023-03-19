package pixivfanbox

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Returns a defined request header needed to communicate with Pixiv Fanbox's API
func GetPixivFanboxHeaders() map[string]string {
	return map[string]string{
		"Origin":  utils.PIXIV_FANBOX_URL,
		"Referer": utils.PIXIV_FANBOX_URL,
	}
}

// Query Pixiv Fanbox's API based on the slice of post IDs and
// returns a map of urls and a map of GDrive urls to download from.
func (pf *PixivFanboxDl) getPostDetails(config *configs.Config, pixivFanboxDlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	maxConcurrency := utils.MAX_API_CALLS
	postIdsLen := len(pf.PostIds)
	if postIdsLen < maxConcurrency {
		maxConcurrency = postIdsLen
	}
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, postIdsLen)
	errChan := make(chan error, postIdsLen)

	baseMsg := "Getting post details from Pixiv Fanbox [%d/" + fmt.Sprintf("%d]...", postIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting %d post details from Pixiv Fanbox!",
			postIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting %d post details from Pixiv Fanbox.\nPlease refer to the logs for more details.",
			postIdsLen,
		),
		postIdsLen,
	)
	progress.Start()

	useHttp3 := utils.IsHttp3Supported(utils.PIXIV_FANBOX, true)
	url := fmt.Sprintf("%s/post.info", utils.PIXIV_FANBOX_API_URL)
	for _, postId := range pf.PostIds {
		wg.Add(1)
		go func(postId string) {
			defer func() {
				<-queue
				wg.Done()
			}()

			queue <- struct{}{}
			header := GetPixivFanboxHeaders()
			params := map[string]string{"postId": postId}
			res, err := request.CallRequest(
				&request.RequestArgs{
					Method:    "GET",
					Url:       url,
					Cookies:   pixivFanboxDlOptions.SessionCookies,
					Headers:   header,
					Params:    params,
					UserAgent: config.UserAgent,
					Http2:     !useHttp3,
					Http3:     useHttp3,
				},
			)
			if err != nil {
				errChan <- fmt.Errorf(
					"pixiv fanbox error %d: failed to get post details for %s, more info => %v",
					utils.CONNECTION_ERROR,
					url,
					err,
				)
			} else if res.StatusCode != 200 {
				errChan <- fmt.Errorf(
					"pixiv fanbox error %d: failed to get post details for %s due to a %s response",
					utils.CONNECTION_ERROR,
					url,
					res.Status,
				)
			} else {
				resChan <- res
			}
			progress.MsgIncrement(baseMsg)
		}(postId)
	}
	wg.Wait()
	close(queue)
	close(resChan)
	close(errChan)

	hasErr := false
	if len(errChan) > 0 {
		hasErr = true
		utils.LogErrors(false, errChan, utils.ERROR)
	}
	progress.Stop(hasErr)
	return processMultiplePostJson(resChan, pixivFanboxDlOptions)
}

func getCreatorPaginatedPosts(creatorId string, config *configs.Config, dlOption *PixivFanboxDlOptions) ([]string, error) {
	params := map[string]string{"creatorId": creatorId}
	headers := GetPixivFanboxHeaders()
	url := fmt.Sprintf(
		"%s/post.paginateCreator",
		utils.PIXIV_FANBOX_API_URL,
	)
	useHttp3 := utils.IsHttp3Supported(utils.PIXIV_FANBOX, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Method:    "GET",
			Url:       url,
			Cookies:   dlOption.SessionCookies,
			Headers:   headers,
			Params:    params,
			UserAgent: config.UserAgent,
			Http2:     !useHttp3,
			Http3:     useHttp3,
		},
	)
	if err != nil || res.StatusCode != 200 {
		const errPrefix = "pixiv fanbox error"
		if err != nil {
			err = fmt.Errorf(
				"%s %d: failed to get creator's posts for %s due to %v",
				errPrefix,
				utils.CONNECTION_ERROR,
				creatorId,
				err,
			)
		} else {
			res.Body.Close()
			err = fmt.Errorf(
				"%s %d: failed to get creator's posts for %s due to %s response",
				errPrefix,
				utils.RESPONSE_ERROR,
				creatorId,
				res.Status,
			)
		}
		return nil, err
	}

	var resJson models.CreatorPaginatedPostsJson
	err = utils.LoadJsonFromResponse(res, &resJson)
	if err != nil {
		return nil, err
	}
	return resJson.Body, nil
}

type resStruct struct {
	json *models.FanboxCreatorPostsJson
	err  error
}

// GetFanboxCreatorPosts returns a slice of post IDs for a given creator
func getFanboxPosts(creatorId, pageNum string, config *configs.Config, dlOption *PixivFanboxDlOptions) ([]string, error) {
	paginatedUrls, err := getCreatorPaginatedPosts(creatorId, config, dlOption)
	if err != nil {
		return nil, err
	}

	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, err
	}

	useHttp3 := utils.IsHttp3Supported(utils.PIXIV_FANBOX, true)
	headers := GetPixivFanboxHeaders()
	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if len(paginatedUrls) < maxConcurrency {
		maxConcurrency = len(paginatedUrls)
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *resStruct, len(paginatedUrls))
	for idx, paginatedUrl := range paginatedUrls {
		curPage := idx + 1
		if curPage < minPage {
			continue
		}
		if hasMax && curPage > maxPage {
			break
		}

		wg.Add(1)
		go func(reqUrl string) {
			defer func() {
				<-queue
				wg.Done()
			}()
			queue <- struct{}{}
			res, err := request.CallRequest(
				&request.RequestArgs{
					Method:    "GET",
					Url:       reqUrl,
					Cookies:   dlOption.SessionCookies,
					Headers:   headers,
					UserAgent: config.UserAgent,
					Http2:     !useHttp3,
					Http3:     useHttp3,
				},
			)
			if err != nil || res.StatusCode != 200 {
				if err == nil {
					res.Body.Close()
				}
				utils.LogError(
					err,
					fmt.Sprintf("failed to get post for %s", reqUrl),
					false,
					utils.ERROR,
				)
				return
			}

			var resJson *models.FanboxCreatorPostsJson
			err = utils.LoadJsonFromResponse(res, &resJson)
			if err != nil {
				resChan <- &resStruct{err: err}
			} else {
				resChan <- &resStruct{json: resJson}
			}
		}(paginatedUrl)
	}
	wg.Wait()
	close(queue)
	close(resChan)

	// parse the JSON response
	var errSlice []error
	var postIds []string
	for res := range resChan {
		if res.err != nil {
			errSlice = append(errSlice, res.err)
			continue
		}

		for _, postInfoMap := range res.json.Body.Items {
			postIds = append(postIds, postInfoMap.Id)
		}
	}

	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	return postIds, nil
}

// Retrieves all the posts based on the slice of creator IDs and updates its slice of post IDs accordingly
func (pf *PixivFanboxDl) getCreatorsPosts(config *configs.Config, dlOptions *PixivFanboxDlOptions) {
	creatorIdsLen := len(pf.CreatorIds)
	if creatorIdsLen != len(pf.CreatorPageNums) {
		panic(
			fmt.Errorf(
				"pixiv fanbox error %d: length of creator IDs and page numbers are not equal",
				utils.DEV_ERROR,
			),
		)
	}

	var errSlice []error
	baseMsg := "Getting post ID(s) from creator(s) on Pixiv Fanbox [%d/" + fmt.Sprintf("%d]...", creatorIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting post ID(s) from %d creator(s) on Pixiv Fanbox!",
			creatorIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting post IDs from %d creator(s) on Pixiv Fanbox!\nPlease refer to logs for more details.",
			creatorIdsLen,
		),
		creatorIdsLen,
	)
	progress.Start()
	for idx, creatorId := range pf.CreatorIds {
		retrievedPostIds, err := getFanboxPosts(
			creatorId,
			pf.CreatorPageNums[idx],
			config,
			dlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			pf.PostIds = append(pf.PostIds, retrievedPostIds...)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)
	pf.PostIds = utils.RemoveSliceDuplicates(pf.PostIds)
}
