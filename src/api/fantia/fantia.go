package fantia

import (
	"fmt"
	"sync"
	"strconv"
	"net/http"
	"encoding/json"
	"path/filepath"
	"github.com/PuerkitoBio/goquery"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
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
		Status string `json:"status"`
		PostContents []struct {
			// Any attachments such as pdfs that are on their dedicated section
			AttachmentURI string `json:"attachment_uri"`

			// For images that are uploaded to their own section
			PostContentPhotos []struct {
				ID int `json:"id"`
				URL struct {
					Original string `json:"original"`
				} `json:"url"`
			} `json:"post_content_photos"`

			// For images that are embedded in the post content
			Comment string `json:"comment"`

			// for attachments such as pdfs that are embedded in the post content
			DownloadUri string `json:"download_uri"`
			Filename string `json:"filename"`
		} `json:"post_contents"`
	} `json:"post"`
}

// Process the JSON response from Fantia's API and 
// returns a map of urls to download from
func ProcessFantiaPost(
	res *http.Response, downloadPath string, 
	downloadThumbnail, downloadImages, downloadAttachments bool,
) []map[string]string {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	var postJson FantiaPost
	resBody := utils.ReadResBody(res)
	err := json.Unmarshal(resBody, &postJson)
	if err != nil {
		utils.LogError(err, fmt.Sprintf("failed to unmarshal json for fantia post\nJSON: %s", string(resBody)), false)
		return nil
	}

	postStruct := postJson.Post
	postId := strconv.Itoa(postStruct.ID)
	postTitle := postStruct.Title
	creatorName := postStruct.Fanclub.User.Name
	postFolderPath := utils.GetPostFolder(filepath.Join(downloadPath, api.FantiaTitle), creatorName, postId, postTitle)

	var urlsMap []map[string]string
	thumbnail := postStruct.Thumb.Original
	if downloadThumbnail && thumbnail != "" {
		urlsMap = append(urlsMap, map[string]string{
			"url":      thumbnail,
			"filepath": postFolderPath,
		})
	}

	postContent := postStruct.PostContents
	if postContent == nil {
		return urlsMap
	}
	for _, content := range postContent{
		if downloadImages {
			// download images that are uploaded to their own section
			postContentPhotos := content.PostContentPhotos
			for _, image := range postContentPhotos {
				imageUrl := image.URL.Original
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, api.ImagesFolder),
				})
			}

			// for images that are embedded in the post content
			comment := content.Comment
			matchedStr := utils.FANTIA_IMAGE_URL_REGEX.FindAllStringSubmatch(comment, -1)
			for _, matched := range matchedStr {
				imageUrl := FantiaUrl + matched[utils.FANTIA_REGEX_URL_INDEX]
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, api.ImagesFolder),
				})
			}
		}

		if downloadAttachments {
			// get the attachment url string if it exists
			attachmentUrl := content.AttachmentURI
			if attachmentUrl != "" {
				attachmentUrlStr := FantiaUrl + attachmentUrl
				urlsMap = append(urlsMap, map[string]string{
					"url":      attachmentUrlStr,
					"filepath": filepath.Join(postFolderPath, api.AttachmentFolder),
				})
			} else if content.DownloadUri != "" {
				// if the attachment url string does not exist, 
				// then get the download url for the file
				downloadUrl := FantiaUrl + content.DownloadUri
				filename := content.Filename
				urlsMap = append(urlsMap, map[string]string{
					"url":      downloadUrl,
					"filepath": filepath.Join(postFolderPath, api.AttachmentFolder, filename),
				})
			}
		}
	}
	return urlsMap
}

// Query Fantia's API based on the slice of post IDs and returns a map of urls to download from
func GetPostDetails(
	postIds []string, cookies []http.Cookie,
	downloadThumbnail, downloadImages, downloadAttachments bool,
) []map[string]string {
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

			header := map[string]string{"Referer": FantiaUrl + "/posts/" + postId}
			res, err := request.CallRequest("GET", url + postId, 30, cookies, header, nil, false)
			if err != nil || res.StatusCode != 200 {
				utils.LogError(err, fmt.Sprintf("failed to get post details for %s", url), false)
			} else {
				resChan <-res
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
		urlsMap = append(urlsMap, ProcessFantiaPost(
				res, utils.DOWNLOAD_PATH, 
				downloadThumbnail, downloadImages, downloadAttachments,
			)...,
		)
		bar.Add(1)
	}
	return urlsMap
}

// Get all the creator's posts by using goquery to parse the HTML response to get the post IDs
func GetFantiaPosts(creatorId string, cookies []http.Cookie) []string {
	var postIds []string
	pageNum := 1
	for {
		url := fmt.Sprintf(FantiaUrl + "/fanclubs/%s/posts", creatorId)
		params := map[string]string{
			"page":   fmt.Sprintf("%d", pageNum),
			"q[s]":   "newer",
			"q[tag]": "",
		}
		res, err := request.CallRequest("GET", url, 30, cookies, nil, params, false)
		if err != nil || res.StatusCode != 200 {
			if err == nil {
				res.Body.Close()
			}
			utils.LogError(err, fmt.Sprintf("failed to get creator's pages for %s", url), false)
			return []string{}
		}

		// parse the response
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			res.Body.Close()
			panic(err)
		}

		// get the post ids similar to using the xpath of //a[@class='link-block']
		hasPosts := false
		doc.Find("a.link-block").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				postIds = append(postIds, utils.GetLastPartOfURL(href))
				hasPosts = true
			} else {
				panic("failed to get href attribute for fantia post, please report this issue!")
			}
		})
		res.Body.Close()

		pageNum++
		// if there are no more posts, break
		if !hasPosts {
			break
		}
	}
	return postIds
}

// Retrieves all the posts based on the slice of creator IDs and returns a slice of post IDs
func GetCreatorsPosts(creatorIds []string, cookies []http.Cookie) []string {
	var postIds []string
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
	for _, creatorId := range creatorIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(creatorId string) {
			defer wg.Done()
			resChan <- GetFantiaPosts(creatorId, cookies)
			bar.Add(1)
			<-queue
		}(creatorId)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	for postIdsRes := range resChan {
		postIds = append(postIds, postIdsRes...)
	}
	return postIds
}