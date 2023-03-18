package fantia

import (
	"errors"
	"fmt"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/PuerkitoBio/goquery"
)

// Query Fantia's API based on the slice of post IDs and get a map of urls to download from.
//
// Note that only the downloading of the URL(s) is/are executed concurrently
// to reduce the chance of the signed AWS S3 URL(s) from expiring before the download is
// executed or completed due to a download queue to avoid resource exhaustion of the user's system.
func (f *FantiaDl) dlFantiaPosts(fantiaDlOptions *FantiaDlOptions, config *configs.Config) {
	var errSlice []error
	postIdsLen := len(f.PostIds)
	url := utils.FANTIA_URL + "/api/v1/posts/"
	for i, postId := range f.PostIds {
		count := i + 1
		msgSuffix := fmt.Sprintf(
			"[%d/%d]",
			count,
			postIdsLen,
		)
		// Now that we have the post ID, we can query Fantia's API
		// to get the post's contents from the JSON response.
		progress := spinner.New(
			spinner.REQ_SPINNER,
			"fgHiYellow",
			fmt.Sprintf(
				"Getting post %s's contents from Fantia %s...",
				postId,
				msgSuffix,
			),
			fmt.Sprintf(
				"Finished getting post %s's contents from Fantia %s!",
				postId,
				msgSuffix,
			),
			fmt.Sprintf(
				"Something went wrong while getting post %s's cotents from Fantia %s.\nPlease refer to the logs for more details.",
				postId,
				msgSuffix,
			),
			postIdsLen,
		)
		progress.Start()

		postApiUrl := url + postId
		header := map[string]string{
			"Referer":      fmt.Sprintf("%s/posts/%s", utils.FANTIA_URL, postId),
			"x-csrf-token": fantiaDlOptions.CsrfToken,
		}
		res, err := request.CallRequest(
			&request.RequestArgs{
				Method:    "GET",
				Url:       postApiUrl,
				Cookies:   fantiaDlOptions.SessionCookies,
				Headers:   header,
				Http2:     true,
				UserAgent: config.UserAgent,
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

			errSlice = append(errSlice, err)
			progress.Stop(true)
			continue
		}
		progress.Stop(false)

		// Process the JSON response to get the urls to download
		progress = spinner.New(
			spinner.JSON_SPINNER,
			"fgHiYellow",
			fmt.Sprintf(
				"Processing retrieved JSON for post %s from Fantia %s...",
				postId,
				msgSuffix,
			),
			fmt.Sprintf(
				"Finished processing retrieved JSON for post %s from Fantia %s!",
				postId,
				msgSuffix,
			),
			fmt.Sprintf(
				"Something went wrong while processing retrieved JSON for post %s from Fantia %s.\nPlease refer to the logs for more details.",
				postId,
				msgSuffix,
			),
			postIdsLen,
		)
		progress.Start()
		urlsToDownload, err := processFantiaPost(
			res,
			utils.DOWNLOAD_PATH,
			fantiaDlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
			progress.Stop(true)
		}
		progress.Stop(false)

		// Download the urls
		request.DownloadUrls(
			urlsToDownload,
			&request.DlOptions{
				MaxConcurrency: utils.MAX_CONCURRENT_DOWNLOADS,
				Headers:        nil,
				Cookies:        fantiaDlOptions.SessionCookies,
				UseHttp3:       false,
			},
			config,
		)
		fmt.Println()
	}

	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
}

// Get all the creator's posts by using goquery to parse the HTML response to get the post IDs
func getCreatorPosts(creatorId, pageNum string, config *configs.Config, dlOption *FantiaDlOptions) ([]string, error) {
	var postIds []string
	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, err
	}

	curPage := minPage
	for {
		url := fmt.Sprintf("%s/fanclubs/%s/posts", utils.FANTIA_URL, creatorId)
		params := map[string]string{
			"page":   fmt.Sprintf("%d", curPage),
			"q[s]":   "newer",
			"q[tag]": "",
		}

		// note that even if the max page is more than
		// the actual number of pages, the response will still be 200 OK.
		res, err := request.CallRequest(
			&request.RequestArgs{
				Method:      "GET",
				Url:         url,
				Cookies:     dlOption.SessionCookies,
				Params:      params,
				Http3:       true,
				CheckStatus: true,
				UserAgent:   config.UserAgent,
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

		// parse the response
		doc, err := goquery.NewDocumentFromReader(res.Body)
		res.Body.Close()
		if err != nil {
			err = fmt.Errorf(
				"fantia error %d, failed to parse response body when getting CSRF token from Fantia: %w",
				utils.HTML_ERROR,
				err,
			)
			return nil, err
		}

		// get the post ids similar to using the xpath of //a[@class='link-block']
		hasPosts := false
		hasHtmlErr := false
		doc.Find("a.link-block").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				postIds = append(postIds, utils.GetLastPartOfUrl(href))
				hasPosts = true
			} else {
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

		// if there are no more posts, break
		if !hasPosts || (hasMax && curPage >= maxPage) {
			break
		}
		curPage++
	}
	return postIds, nil
}

// Retrieves all the posts based on the slice of creator IDs and updates its PostIds slice
func (f *FantiaDl) getCreatorsPosts(config *configs.Config, dlOption *FantiaDlOptions) {
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
				<-queue
				wg.Done()
			}()

			queue <- struct{}{}
			postIds, err := getCreatorPosts(
				creatorId,
				f.FanclubPageNums[pageNumIdx],
				config,
				dlOption,
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
