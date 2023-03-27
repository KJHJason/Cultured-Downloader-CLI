package fantia

import (
	"errors"
	"fmt"
	"sync"
	"strconv"
	"net/http"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/PuerkitoBio/goquery"
)

type fantiaPostArgs struct {
	msgSuffix  string
	postId     string
	url        string
	postIdsLen int
}

func getFantiaPostDetails(postArg *fantiaPostArgs, dlOptionss *FantiaDlOptions) (*http.Response, error) {
	// Now that we have the post ID, we can query Fantia's API
	// to get the post's contents from the JSON response.
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			"Getting post %s's contents from Fantia %s...",
			postArg.postId,
			postArg.msgSuffix,
		),
		fmt.Sprintf(
			"Finished getting post %s's contents from Fantia %s!",
			postArg.postId,
			postArg.msgSuffix,
		),
		fmt.Sprintf(
			"Something went wrong while getting post %s's cotents from Fantia %s.\nPlease refer to the logs for more details.",
			postArg.postId,
			postArg.msgSuffix,
		),
		postArg.postIdsLen,
	)
	progress.Start()

	postApiUrl := postArg.url + postArg.postId
	header := map[string]string{
		"Referer":      fmt.Sprintf("%s/posts/%s", utils.FANTIA_URL, postArg.postId),
		"x-csrf-token": dlOptionss.CsrfToken,
	}
	useHttp3 := utils.IsHttp3Supported(utils.FANTIA, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Method:    "GET",
			Url:       postApiUrl,
			Cookies:   dlOptionss.SessionCookies,
			Headers:   header,
			Http2:     !useHttp3,
			Http3:     useHttp3,
			UserAgent: dlOptionss.Configs.UserAgent,
		},
	)
	if err != nil || res.StatusCode != 200 {
		errCode := utils.CONNECTION_ERROR
		if err == nil {
			errCode = res.StatusCode
		}

		errMsg := fmt.Sprintf(
			"fantia error %d: failed to get post details for %s",
			errCode,
			postApiUrl,
		)
		if err != nil {
			err = fmt.Errorf(
				"%s, more info => %v",
				errMsg,
				err,
			)
		} else {
			err = errors.New(errMsg)
		}

		progress.Stop(true)
		return nil, err
	}

	progress.Stop(false)
	return res, nil
}

// Query Fantia's API based on the slice of post IDs and get a map of urls to download from.
//
// Note that only the downloading of the URL(s) is/are executed concurrently
// to reduce the chance of the signed AWS S3 URL(s) from expiring before the download is
// executed or completed due to a download queue to avoid resource exhaustion of the user's system.
func (f *FantiaDl) dlFantiaPosts(dlOptionss *FantiaDlOptions) []*request.ToDownload {
	var errSlice []error
	var gdriveLinks []*request.ToDownload
	postIdsLen := len(f.PostIds)
	url := utils.FANTIA_URL + "/api/v1/posts/"
	for i, postId := range f.PostIds {
		count := i + 1
		msgSuffix := fmt.Sprintf(
			"[%d/%d]",
			count,
			postIdsLen,
		)

		res, err := getFantiaPostDetails(
			&fantiaPostArgs{
				msgSuffix:  msgSuffix,
				postId:     postId,
				url:        url,
				postIdsLen: postIdsLen,
			},
			dlOptionss,
		)
		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}

		urlsToDownload, postGdriveUrls, err := processIllustDetailApiRes(
			&processIllustArgs{
				res:          res,
				postId:       postId,
				postIdsLen:  postIdsLen,
				msgSuffix:   msgSuffix,
			},
			dlOptionss,
		)
		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}
		gdriveLinks = append(gdriveLinks, postGdriveUrls...)

		// Download the urls
		request.DownloadUrls(
			urlsToDownload,
			&request.DlOptions{
				MaxConcurrency: utils.MAX_CONCURRENT_DOWNLOADS,
				Headers:        nil,
				Cookies:        dlOptionss.SessionCookies,
				UseHttp3:       false,
			},
			dlOptionss.Configs,
		)
		fmt.Println()
	}

	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	return gdriveLinks
}

// Parse the HTML response from the creator's page to get the post IDs.
func parseCreatorHtml(res *http.Response, creatorId string) ([]string, error) {
	// parse the response
	doc, err := goquery.NewDocumentFromReader(res.Body)
	res.Body.Close()
	if err != nil {
		err = fmt.Errorf(
			"fantia error %d, failed to parse response body when getting posts for Fantia Fanclub %s, more info => %v",
			utils.HTML_ERROR,
			creatorId,
			err,
		)
		return nil, err
	}

	// get the post ids similar to using the xpath of //a[@class='link-block']
	hasHtmlErr := false
	var postIds []string
	doc.Find("a.link-block").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			postIds = append(postIds, utils.GetLastPartOfUrl(href))
		} else if !hasHtmlErr {
			hasHtmlErr = true
		}
	})

	if hasHtmlErr {
		return nil, fmt.Errorf(
			"fantia error %d, failed to get href attribute for Fantia Fanclub %s, please report this issue",
			utils.HTML_ERROR,
			creatorId,
		)
	}
	return postIds, nil
}

// Get all the creator's posts by using goquery to parse the HTML response to get the post IDs
func getCreatorPosts(creatorId, pageNum string, dlOptions *FantiaDlOptions) ([]string, error) {
	var postIds []string
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, err
	}

	useHttp3 := utils.IsHttp3Supported(utils.FANTIA, false)
	curPage := minPage
	for {
		url := fmt.Sprintf("%s/fanclubs/%s/posts", utils.FANTIA_URL, creatorId)
		params := map[string]string{
			"page":   strconv.Itoa(curPage),
			"q[s]":   "newer",
			"q[tag]": "",
		}

		// note that even if the max page is more than
		// the actual number of pages, the response will still be 200 OK.
		res, err := request.CallRequest(
			&request.RequestArgs{
				Method:      "GET",
				Url:         url,
				Cookies:     dlOptions.SessionCookies,
				Params:      params,
				Http2:       !useHttp3,
				Http3:       useHttp3,
				CheckStatus: true,
				UserAgent:   dlOptions.Configs.UserAgent,
			},
		)
		if err != nil {
			err = fmt.Errorf(
				"fantia error %d: failed to get creator's pages for %s, more info => %v",
				utils.CONNECTION_ERROR,
				url,
				err,
			)
			return nil, err
		}

		creatorPostIds, err := parseCreatorHtml(res, creatorId)
		if err != nil {
			return nil, err
		}
		postIds = append(postIds, creatorPostIds...)

		// if there are no more posts, break
		if len(creatorPostIds) == 0 || (hasMax && curPage >= maxPage) {
			break
		}
		curPage++
	}
	return postIds, nil
}

// Retrieves all the posts based on the slice of creator IDs and updates its PostIds slice
func (f *FantiaDl) getCreatorsPosts(dlOptions *FantiaDlOptions) {
	creatorIdsLen := len(f.FanclubIds)
	if creatorIdsLen != len(f.FanclubPageNums) {
		panic(
			fmt.Errorf(
				"fantia error %d: creator IDs and page numbers slices are not the same length",
				utils.DEV_ERROR,
			),
		)
	}

	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if creatorIdsLen < maxConcurrency {
		maxConcurrency = creatorIdsLen
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan []string, creatorIdsLen)
	errChan := make(chan error, creatorIdsLen)

	baseMsg := "Getting post ID(s) from Fanclubs(s) on Fantia [%d/" + fmt.Sprintf("%d]...", creatorIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting post ID(s) from %d Fanclubs(s) on Fantia!",
			creatorIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting post IDs from %d Fanclubs(s) on Fantia.\nPlease refer to the logs for more details.",
			creatorIdsLen,
		),
		creatorIdsLen,
	)
	progress.Start()
	for idx, creatorId := range f.FanclubIds {
		wg.Add(1)
		go func(creatorId string, pageNumIdx int) {
			defer func() {
				wg.Done()
				<-queue
			}()

			queue <- struct{}{}
			postIds, err := getCreatorPosts(
				creatorId,
				f.FanclubPageNums[pageNumIdx],
				dlOptions,
			)
			if err != nil {
				errChan <- err
			} else {
				resChan <- postIds
			}

			progress.MsgIncrement(baseMsg)
		}(creatorId, idx)
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

	for postIdsRes := range resChan {
		f.PostIds = append(f.PostIds, postIdsRes...)
	}
	f.PostIds = utils.RemoveSliceDuplicates(f.PostIds)
}
