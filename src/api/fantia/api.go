package fantia

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
	"os"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/fatih/color"
)

type fantiaPostArgs struct {
	msgSuffix  string
	postId     string
	url        string
	postIdsLen int
}

func getFantiaPostDetails(postArg *fantiaPostArgs, dlOptions *FantiaDlOptions) (*http.Response, error) {
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
		"x-csrf-token": dlOptions.CsrfToken,
	}
	useHttp3 := utils.IsHttp3Supported(utils.FANTIA, true)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Method:    "GET",
			Url:       postApiUrl,
			Cookies:   dlOptions.SessionCookies,
			Headers:   header,
			Http2:     !useHttp3,
			Http3:     useHttp3,
			UserAgent: dlOptions.Configs.UserAgent,
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

const captchaBtnSelector = `//input[@name='commit']`

// Automatically try to solve the reCAPTCHA for Fantia.
func autoSolveCaptcha(dlOptions *FantiaDlOptions) error {
	actions := []chromedp.Action{
		utils.SetChromedpAllocCookies(dlOptions.SessionCookies),
		chromedp.Navigate(utils.FANTIA_RECAPTCHA_URL),
		chromedp.WaitVisible(captchaBtnSelector, chromedp.BySearch),
		chromedp.Click(captchaBtnSelector, chromedp.BySearch),
		chromedp.WaitVisible(`//h3[@class='mb-15'][contains(text(), 'ファンティアでクリエイターを応援しよう！')]`, chromedp.BySearch),
	}

	allocCtx, cancel := utils.GetDefaultChromedpAlloc(dlOptions.Configs.UserAgent)
	defer cancel()

	allocCtx, cancel = context.WithTimeout(allocCtx, 45 * time.Second)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		"Solving reCAPTCHA for Fantia...",
		"Successfully solved reCAPTCHA for Fantia!",
		"",
		0,
	)
	progress.Start()
	if err := utils.ExecuteChromedpActions(allocCtx, cancel, actions...); err != nil {
		var fmtErr error
		if errors.Is(err, context.DeadlineExceeded) {
			fmtErr = fmt.Errorf(
				"fantia error %d: failed to solve reCAPTCHA for Fantia due to timeout, please visit %s to solve it manually and try again", 
				utils.CAPTCHA_ERROR,
				utils.FANTIA_RECAPTCHA_URL,
			)
		} else {
			fullErr := fmt.Errorf("fantia error %d: failed to solve reCAPTCHA for Fantia, more info => %v", utils.CAPTCHA_ERROR, err)
			utils.LogError(fullErr, "", false, utils.ERROR)
		}

		progress.ErrMsg = fmtErr.Error() + "\n"
		progress.Stop(true)
		return fmtErr
	}
	progress.Stop(false)
	return nil
}

// Manually ask the user to solve the reCAPTCHA on Fantia for the current session.
func manualSolveCaptcha(dlOptions *FantiaDlOptions) error {
	// Check if the reCAPTCHA has been solved.
	// If it has, we can continue with the download.
	useHttp3 := utils.IsHttp3Supported(utils.FANTIA, true)
	instructions := fmt.Sprintf(
		"Please solve the reCAPTCHA on Fantia at %s with the SAME session to continue.",
		utils.FANTIA_RECAPTCHA_URL,
	)
	utils.AlertWithoutErr(utils.Title, instructions)

	for {
		color.Yellow(instructions)
		color.Yellow("Please press ENTER when you're done.")
		fmt.Scanln()

		_, err := request.CallRequest(
			&request.RequestArgs{
				Method:      "GET",
				Url:         utils.FANTIA_URL + "/mypage/users/plans",
				Cookies:     dlOptions.SessionCookies,
				Http2:       !useHttp3,
				Http3:       useHttp3,
				UserAgent:   dlOptions.Configs.UserAgent,
				CheckStatus: true,
			},
		)
		if err != nil {
			color.Red("failed to check if reCAPTCHA has been solved\n%v", err)
			continue
		}
		return nil
	}
}

func SolveCaptcha(dlOptions *FantiaDlOptions, alertUser bool) error {
	if alertUser {
		color.Yellow("\nWarning: reCAPTCHA detected for the current Fantia session...")
	}

	if len(dlOptions.SessionCookies) == 0 {
		// Since reCAPTCHA is per session, the program shall avoid 
		// trying to solve it and alert the user to login or create a Fantia account.
		// It is possible that the reCAPTCHA is per IP address for guests, but I'm not sure.
		color.Red(
			fmt.Sprintf(
				"fantia error %d: reCAPTCHA detected but you are not logged in. Please login to Fantia and try again.",
				utils.CAPTCHA_ERROR,
			),
		)
		os.Exit(1)
	}

	if dlOptions.AutoSolveCaptcha {
		return autoSolveCaptcha(dlOptions)
	}
	return manualSolveCaptcha(dlOptions)
}

// try the alternative method if the first one fails.
//
// E.g. User preferred to solve the reCAPTCHA automatically, but the program failed to do so,
//      The program will then ask the user to solve the reCAPTCHA manually on their browser with the SAME session.
func handleCaptchaErr(err error, dlOptions *FantiaDlOptions, alertUser bool) error {
	if err == nil {
		return nil
	}

	dlOptions.AutoSolveCaptcha = !dlOptions.AutoSolveCaptcha
	return SolveCaptcha(dlOptions, alertUser)
}

const fantiaPostUrl = utils.FANTIA_URL + "/api/v1/posts/"
func dlFantiaPost(count, maxCount int, postId string, dlOptions *FantiaDlOptions) ([]*request.ToDownload, error) {
	msgSuffix := fmt.Sprintf(
		"[%d/%d]",
		count,
		maxCount,
	)

	res, err := getFantiaPostDetails(
		&fantiaPostArgs{
			msgSuffix:  msgSuffix,
			postId:     postId,
			url:        fantiaPostUrl,
			postIdsLen: maxCount,
		},
		dlOptions,
	)
	if err != nil {
		return nil, err
	}

	urlsToDownload, postGdriveUrls, err := processIllustDetailApiRes(
		&processIllustArgs{
			res:          res,
			postId:       postId,
			postIdsLen:   maxCount,
			msgSuffix:    msgSuffix,
		},
		dlOptions,
	)
	if err == errRecaptcha {
		err = SolveCaptcha(dlOptions, true)
		if err != nil {
			if err := handleCaptchaErr(err, dlOptions, true); err != nil {
				os.Exit(1)
			}
		}

		return dlFantiaPost(count, maxCount, postId, dlOptions)
	} else if err != nil {
		return nil, err
	}

	// Download the urls
	request.DownloadUrls(
		urlsToDownload,
		&request.DlOptions{
			MaxConcurrency: utils.MAX_CONCURRENT_DOWNLOADS,
			Headers:        nil,
			Cookies:        dlOptions.SessionCookies,
			UseHttp3:       false,
		},
		dlOptions.Configs,
	)
	fmt.Println()
	return postGdriveUrls, nil
}

// Query Fantia's API based on the slice of post IDs and get a map of urls to download from.
//
// Note that only the downloading of the URL(s) is/are executed concurrently
// to reduce the chance of the signed AWS S3 URL(s) from expiring before the download is
// executed or completed due to a download queue to avoid resource exhaustion of the user's system.
func (f *FantiaDl) dlFantiaPosts(dlOptions *FantiaDlOptions) []*request.ToDownload {
	var errSlice []error
	var gdriveLinks []*request.ToDownload
	postIdsLen := len(f.PostIds)
	for i, postId := range f.PostIds {
		postGdriveLinks, err := dlFantiaPost(i+1, postIdsLen, postId, dlOptions)

		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}
		if len(postGdriveLinks) > 0 {
			gdriveLinks = append(gdriveLinks, postGdriveLinks...)
		}
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
