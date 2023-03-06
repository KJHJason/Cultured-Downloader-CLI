package fantia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/PuerkitoBio/goquery"
)

const FantiaUrl = "https://fantia.jp"

type FantiaPost struct {
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
func ProcessFantiaPost(res *http.Response, downloadPath string, fantiaDlOptions *FantiaDlOptions) []map[string]string {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	var postJson FantiaPost
	resBody, err := utils.ReadResBody(res)
	if err != nil {
		utils.LogError(err, "failed to read response body for fantia post", false)
		return nil
	}

	err = json.Unmarshal(resBody, &postJson)
	if err != nil {
		utils.LogError(err, fmt.Sprintf("failed to unmarshal json for fantia post\nJSON: %s", string(resBody)), false)
		return nil
	}

	postStruct := postJson.Post
	postId := strconv.Itoa(postStruct.ID)
	postTitle := postStruct.Title
	creatorName := postStruct.Fanclub.User.Name
	postFolderPath := utils.GetPostFolder(filepath.Join(downloadPath, utils.FANTIA_TITLE), creatorName, postId, postTitle)

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
		return urlsMap
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
				imageUrl := FantiaUrl + matched[utils.FANTIA_REGEX_URL_INDEX]
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
				attachmentUrlStr := FantiaUrl + attachmentUrl
				urlsMap = append(urlsMap, map[string]string{
					"url":      attachmentUrlStr,
					"filepath": filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER),
				})
			} else if content.DownloadUri != "" {
				// if the attachment url string does not exist,
				// then get the download url for the file
				downloadUrl := FantiaUrl + content.DownloadUri
				filename := content.Filename
				urlsMap = append(urlsMap, map[string]string{
					"url":      downloadUrl,
					"filepath": filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename),
				})
			}
		}
	}
	return urlsMap
}

// Query Fantia's API based on the slice of post IDs and returns a map of urls to download from
func GetPostDetails(postIds []string, fantiaDlOptions *FantiaDlOptions) []map[string]string {
	maxConcurrency := utils.MAX_API_CALLS
	if len(postIds) < maxConcurrency {
		maxConcurrency = len(postIds)
	}
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(postIds))

	bar := utils.GetProgressBar(
		len(postIds),
		"Getting Fantia post details...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting %d Fantia post details!", len(postIds))),
	)
	url := FantiaUrl + "/api/v1/posts/"
	for _, postId := range postIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(postId string) {
			defer wg.Done()

			postApiUrl := url + postId
			header := map[string]string{
				"Referer":      fmt.Sprintf("%s/posts/%s", FantiaUrl, postId),
				"x-csrf-token": fantiaDlOptions.CsrfToken,
			}
			res, err := request.CallRequest(
				"GET",
				postApiUrl,
				30,
				fantiaDlOptions.SessionCookies,
				header,
				nil,
				false,
			)
			if err != nil || res.StatusCode != 200 {
				utils.LogError(err, fmt.Sprintf("failed to get post details for %s", postApiUrl), false)
			} else {
				resChan <- res
			}
			bar.Add(1)
			<-queue
		}(postId)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	// parse the responses
	bar = utils.GetProgressBar(
		len(resChan),
		"Processing received JSON(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished processing %d JSON(s)!", len(resChan))),
	)
	var urlsMap []map[string]string
	for res := range resChan {
		urlsSlice := ProcessFantiaPost(
			res, utils.DOWNLOAD_PATH,
			fantiaDlOptions,
		)
		urlsMap = append(urlsMap, urlsSlice...)
		bar.Add(1)
	}
	return urlsMap
}

// Get all the creator's posts by using goquery to parse the HTML response to get the post IDs
func GetFantiaPosts(creatorId string, cookies []http.Cookie) ([]string, error) {
	var postIds []string
	pageNum := 1
	for {
		url := fmt.Sprintf("%s/fanclubs/%s/posts", FantiaUrl, creatorId)
		params := map[string]string{
			"page":   fmt.Sprintf("%d", pageNum),
			"q[s]":   "newer",
			"q[tag]": "",
		}
		res, err := request.CallRequest("GET", url, 30, cookies, nil, params, true)
		if err != nil {
			utils.LogError(err, fmt.Sprintf("failed to get creator's pages for %s", url), false)
			return []string{}, nil
		}

		// parse the response
		doc, err := goquery.NewDocumentFromReader(res.Body)
		res.Body.Close()
		if err != nil {
			err = fmt.Errorf("error %d, failed to parse response body when getting CSRF token from Fantia: %w", utils.HTML_ERROR, err)
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
				"error %d, failed to get href attribute for Fantia Fanclub %s, please report this issue",
				utils.HTML_ERROR,
				creatorId,
			)
		}

		pageNum++
		// if there are no more posts, break
		if !hasPosts {
			break
		}
	}
	return postIds, nil
}

// Retrieves all the posts based on the slice of creator IDs and returns a slice of post IDs
func GetCreatorsPosts(creatorIds []string, cookies []http.Cookie) []string {
	bar := utils.GetProgressBar(
		len(creatorIds),
		"Getting post IDs from creator(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting post IDs from %d creator(s)!", len(creatorIds))),
	)

	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if len(creatorIds) < maxConcurrency {
		maxConcurrency = len(creatorIds)
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan []string, len(creatorIds))
	errChan := make(chan error, len(creatorIds))
	for _, creatorId := range creatorIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(creatorId string) {
			defer wg.Done()
			postIds, err := GetFantiaPosts(creatorId, cookies)
			if err != nil {
				errChan <- err
			} else {
				resChan <- postIds
			}

			bar.Add(1)
			<-queue
		}(creatorId)
	}
	close(queue)
	wg.Wait()
	close(resChan)
	close(errChan)

	utils.LogErrors(false, &errChan)
	var postIds []string
	for postIdsRes := range resChan {
		postIds = append(postIds, postIdsRes...)
	}
	return postIds
}
