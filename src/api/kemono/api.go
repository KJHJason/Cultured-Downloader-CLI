package kemono

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/PuerkitoBio/goquery"
)

type kemonoChanRes struct {
	urlsToDownload []*request.ToDownload
	gdriveLinks    []*request.ToDownload
	err            error
}

func getKemonoPartyHeaders(tld string) map[string]string {
	return map[string]string{
		"Host": getKemonoUrl(tld),
	}
}

func getKemonoUrl(tld string) string {
	if tld == utils.KEMONO_TLD {
		return utils.KEMONO_URL
	}
	return utils.BACKUP_KEMONO_URL
}

func getKemonoApiUrl(tld string) string {
	if tld == utils.KEMONO_TLD {
		return utils.KEMONO_API_URL
	}
	return utils.BACKUP_KEMONO_API_URL
}

func getKemonoUrlFromConditions(isBackup, isApi bool) string {
	if isApi {
		if isBackup {
			return utils.BACKUP_KEMONO_API_URL
		}
		return utils.KEMONO_API_URL
	}

	if isBackup {
		return utils.BACKUP_KEMONO_URL
	}
	return utils.KEMONO_URL
}

var errSessionCookieNotFound = errors.New("could not find session cookie")
func getKemonoUrlFromCookie(cookie []*http.Cookie, isApi bool) (string, string, error) {
	for _, c := range cookie {
		if c.Name == utils.KEMONO_SESSION_COOKIE_NAME {
			if c.Domain == utils.KEMONO_COOKIE_DOMAIN {
				return getKemonoUrlFromConditions(false, isApi), utils.KEMONO_TLD ,nil
			} else {
				return getKemonoUrlFromConditions(true, isApi), utils.KEMONO_BACKUP_TLD, nil
			}
		}
	}
	return "", "", errSessionCookieNotFound
}

// To obtain the creator's username
func parseCreatorHtml(res *http.Response, url string) (string, error) {
	// parse the response
	doc, err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()
	if err != nil {
		err = fmt.Errorf(
			"kemono error %d, failed to parse response body when getting creator name from Kemono Party at %s\nmore info => %v",
			utils.HTML_ERROR,
			url,
			err,
		)
		return "", err
	}

	// <span itemprop="name">creator-name</span> => creator-name
	creatorName := doc.Find("span[itemprop=name]").Text()
	if creatorName == "" {
		return "", fmt.Errorf(
			"kemono error %d, failed to get creator name from Kemono Party at %s\nplease report this issue",
			utils.HTML_ERROR,
			url,
		)
	}

	return creatorName, nil
}

var creatorNameCacheLock sync.Mutex
var creatorNameCache = make(map[string]string)
func getCreatorName(service, userId string, dlOptions *KemonoDlOptions) (string, error) {
	creatorNameCacheLock.Lock()
	defer creatorNameCacheLock.Unlock()
	cacheKey := fmt.Sprintf("%s/%s", service, userId)
	if name, ok := creatorNameCache[cacheKey]; ok {
		return name, nil
	}

	apiUrl, tld, err := getKemonoUrlFromCookie(dlOptions.SessionCookies, false)
	if err != nil {
		return userId, err
	}

	useHttp3 := utils.IsHttp3Supported(utils.KEMONO, true)
	url := fmt.Sprintf(
		"%s/%s/user/%s",
		apiUrl,
		service,
		userId,
	)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url:         url,
			Method:      "GET",
			Headers:     getKemonoPartyHeaders(tld),
			UserAgent:   dlOptions.Configs.UserAgent,
			Cookies:     dlOptions.SessionCookies,
			Http2:       !useHttp3,
			Http3:       useHttp3,
			CheckStatus: true,
		},
	)
	if err != nil {
		return userId, err
	}

	creatorName, err := parseCreatorHtml(res, url)
	if err != nil {
		return userId, err
	}

	creatorNameCache[cacheKey] = creatorName
	return creatorName, nil
}

func getPostDetails(post *models.KemonoPostToDl, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	useHttp3 := utils.IsHttp3Supported(utils.KEMONO, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Url: fmt.Sprintf(
				"%s/%s/user/%s/post/%s",
				getKemonoApiUrl(post.Tld),
				post.Service,
				post.CreatorId,
				post.PostId,
			),
			Method:      "GET",
			Headers:     getKemonoPartyHeaders(post.Tld),
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

	postsToDl, gdriveLinks := processMultipleJson(resJson, post.Tld, downloadPath, dlOptions)
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
					getKemonoApiUrl(creator.Tld),
					creator.Service,
					creator.CreatorId,
				),
				Method:      "GET",
				UserAgent:   dlOptions.Configs.UserAgent,
				Headers:     getKemonoPartyHeaders(creator.Tld),
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

		posts, gdriveLinks := processMultipleJson(resJson, creator.Tld, downloadPath, dlOptions)
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

func processFavCreator(resJson models.KemonoFavCreatorJson, tld string) []*models.KemonoCreatorToDl {
	var creators []*models.KemonoCreatorToDl
	for _, creator := range resJson {
		creators = append(creators, &models.KemonoCreatorToDl{
			CreatorId: creator.Id,
			Service:   creator.Service,
			PageNum:   "", // download all pages
			Tld:       tld,
		})
	}
	return creators
}

func getFavourites(downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	apiUrl, tld, err := getKemonoUrlFromCookie(dlOptions.SessionCookies, true)
	if err != nil {
		return nil, nil, err
	}

	useHttp3 := utils.IsHttp3Supported(utils.KEMONO, true)
	params := map[string]string{
		"type": "artist",
	}
	reqArgs := &request.RequestArgs{
		Url:         fmt.Sprintf("%s/v1/account/favorites", apiUrl),
		Method:      "GET",
		Cookies:     dlOptions.SessionCookies,
		Params:      params,
		Headers:     getKemonoPartyHeaders(tld),
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
	artistToDl := processFavCreator(creatorResJson, tld)

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
	urlsToDownload, gdriveLinks := processMultipleJson(postResJson, tld, downloadPath, dlOptions)

	creatorsPost, creatorsGdrive := getMultipleCreators(artistToDl, downloadPath, dlOptions)
	urlsToDownload = append(urlsToDownload, creatorsPost...)
	gdriveLinks = append(gdriveLinks, creatorsGdrive...)

	return urlsToDownload, gdriveLinks, nil
}
