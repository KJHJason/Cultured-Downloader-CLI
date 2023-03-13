package fantia

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/PuerkitoBio/goquery"
)

type fantiaPost struct {
	Post struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Thumb struct {
			Original string `json:"original"`
		} `json:"thumb"`
		Fanclub struct {
			User struct {
				Name string `json:"name"`
			} `json:"user"`
		} `json:"fanclub"`
		Status       string `json:"status"`
		PostContents []struct {
			// Any attachments such as pdfs that are on their dedicated section
			AttachmentURI string `json:"attachment_uri"`

			// For images that are uploaded to their own section
			PostContentPhotos []struct {
				ID  int `json:"id"`
				URL struct {
					Original string `json:"original"`
				} `json:"url"`
			} `json:"post_content_photos"`

			// For images that are embedded in the post content
			Comment string `json:"comment"`

			// for attachments such as pdfs that are embedded in the post content
			DownloadUri string `json:"download_uri"`
			Filename    string `json:"filename"`
		} `json:"post_contents"`
	} `json:"post"`
}

// Process the JSON response from Fantia's API and
// returns a map of urls to download from
func processFantiaPost(res *http.Response, downloadPath string, fantiaDlOptions *FantiaDlOptions) ([]map[string]string, error) {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	var postJson fantiaPost
	resBody, err := utils.ReadResBody(res)
	if err != nil {
		return nil, fmt.Errorf(
			"%v\nMore details: failed to read response body for fantia post",
			err,
		)
	}

	err = json.Unmarshal(resBody, &postJson)
	if err != nil {
		return nil, fmt.Errorf(
			"fantia error %d: failed to unmarshal json for fantia post\nJSON: %s",
			utils.JSON_ERROR,
			string(resBody),
		)
	}

	postStruct := postJson.Post
	postId := strconv.Itoa(postStruct.ID)
	postTitle := postStruct.Title
	creatorName := postStruct.Fanclub.User.Name
	postFolderPath := utils.GetPostFolder(
		filepath.Join(
			downloadPath,
			utils.FANTIA_TITLE,
		),
		creatorName,
		postId,
		postTitle,
	)

	var urlsMap []map[string]string
	thumbnail := postStruct.Thumb.Original
	if fantiaDlOptions.DlThumbnails && thumbnail != "" {
		urlsMap = append(urlsMap, map[string]string{
			"url":      thumbnail,
			"filepath": postFolderPath,
		})
	}

	postContent := postStruct.PostContents
	if postContent == nil {
		return urlsMap, nil
	}
	for _, content := range postContent {
		if fantiaDlOptions.DlImages {
			// download images that are uploaded to their own section
			postContentPhotos := content.PostContentPhotos
			for _, image := range postContentPhotos {
				imageUrl := image.URL.Original
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
				})
			}

			// for images that are embedded in the post content
			comment := content.Comment
			matchedStr := utils.FANTIA_IMAGE_URL_REGEX.FindAllStringSubmatch(comment, -1)
			for _, matched := range matchedStr {
				imageUrl := utils.FANTIA_URL + matched[utils.FANTIA_REGEX_URL_INDEX]
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
				})
			}
		}

		if fantiaDlOptions.DlAttachments {
			// get the attachment url string if it exists
			attachmentUrl := content.AttachmentURI
			if attachmentUrl != "" {
				attachmentUrlStr := utils.FANTIA_URL + attachmentUrl
				urlsMap = append(urlsMap, map[string]string{
					"url":      attachmentUrlStr,
					"filepath": filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER),
				})
			} else if content.DownloadUri != "" {
				// if the attachment url string does not exist,
				// then get the download url for the file
				downloadUrl := utils.FANTIA_URL + content.DownloadUri
				filename := content.Filename
				urlsMap = append(urlsMap, map[string]string{
					"url":      downloadUrl,
					"filepath": filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename),
				})
			}
		}
	}
	return urlsMap, nil
}

// Query Fantia's API based on the slice of post IDs and get a map of urls to download from.
//
// Note that only the downloading of the URL(s) is/are executed parallelly 
// to reduce the chance of the signed AWS S3 URL(s) from expiring before the download is
// executed or completed due to a download queue to avoid resource exhaustion of the user's system.
func dlFantiaPosts(postIds []string, fantiaDlOptions *FantiaDlOptions, config *api.Config) {
	var errSlice []error
	postIdsLen := len(postIds)
	url := utils.FANTIA_URL + "/api/v1/posts/"
	for i, postId := range postIds {
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
		request.DownloadURLsParallel(
			urlsToDownload,
			&request.DlOptions{
				MaxConcurrency:         utils.MAX_CONCURRENT_DOWNLOADS,
				Headers:                nil,
				Cookies:                fantiaDlOptions.SessionCookies,
				UseHttp3:               false,
				OverwriteExistingFiles: config.OverwriteFiles,
				UserAgent:              config.UserAgent,
			},
		)
		fmt.Println()
	}

	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, errSlice...)
	}
}

// Get all the creator's posts by using goquery to parse the HTML response to get the post IDs
func getFantiaPosts(creatorId, pageNum, userAgent string, cookies []*http.Cookie) ([]string, error) {
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
				Cookies:     cookies,
				Params:      params,
				Http3:       true,
				CheckStatus: true,
				UserAgent:   userAgent,
			},
		)
		if err != nil {
			err = fmt.Errorf(
				"fantia error %d: failed to get creator's pages for %s, more info => %v",
				utils.CONNECTION_ERROR,
				url,
				err,
			)
			return []string{}, err
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
				postIds = append(postIds, utils.GetLastPartOfURL(href))
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

// Retrieves all the posts based on the slice of creator IDs and returns a slice of post IDs
func getCreatorsPosts(creatorIds, pageNums []string, userAgent string, cookies []*http.Cookie) []string {
	creatorIdsLen := len(creatorIds)
	if creatorIdsLen != len(pageNums) {
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
	for idx, creatorId := range creatorIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(creatorId string, pageNumIdx int) {
			defer func() {
				<-queue
				wg.Done()
			}()

			postIds, err := getFantiaPosts(
				creatorId,
				pageNums[pageNumIdx],
				userAgent,
				cookies,
			)
			if err != nil {
				errChan <- err
			} else {
				resChan <- postIds
			}

			progress.MsgIncrement(baseMsg)
		}(creatorId, idx)
	}
	close(queue)
	wg.Wait()
	close(resChan)
	close(errChan)

	hasErr := false
	if len(errChan) > 0 {
		hasErr = true
		utils.LogErrors(false, errChan)
	}
	progress.Stop(hasErr)

	var postIds []string
	for postIdsRes := range resChan {
		postIds = append(postIds, postIdsRes...)
	}
	return postIds
}

// Start the download process for Fantia
func FantiaDownloadProcess(config *api.Config, fantiaDl *FantiaDl, fantiaDlOptions *FantiaDlOptions) {
	if !fantiaDlOptions.DlThumbnails && !fantiaDlOptions.DlImages && !fantiaDlOptions.DlAttachments {
		return
	}

	if len(fantiaDl.PostIds) > 0 {
		dlFantiaPosts(
			fantiaDl.PostIds,
			fantiaDlOptions,
			config,
		)
	}
	if len(fantiaDl.FanclubIds) > 0 {
		fantiaPostIds := getCreatorsPosts(
			fantiaDl.FanclubIds,
			fantiaDl.FanclubPageNums,
			config.UserAgent,
			fantiaDlOptions.SessionCookies,
		)
		dlFantiaPosts(
			fantiaPostIds,
			fantiaDlOptions,
			config,
		)
	}
}
