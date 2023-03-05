package pixivfanbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Returns a defined request header needed to communicate with Pixiv Fanbox's API
func GetPixivFanboxHeaders() map[string]string {
	return map[string]string{
		"Origin":  "https://www.fanbox.cc",
		"Referer": "https://www.fanbox.cc/",
	}
}

// Process and detects for any external download links from the post's text content
func ProcessPixivFanboxText(postBody interface{}, postFolderPath string, downloadGdrive bool) []map[string]string {
	if postBody == nil {
		return nil
	}

	// split the text by newlines
	postBodySlice := strings.FieldsFunc(
		postBody.(string),
		func(c rune) bool {
			return c == '\n'
		},
	)
	loggedPassword := false
	var detectedGdriveLinks []map[string]string
	for _, text := range postBodySlice {
		if utils.DetectPasswordInText(text) && !loggedPassword {
			// Log the entire post text if it contains a password
			filePath := filepath.Join(postFolderPath, api.PasswordFilename)
			if !utils.PathExists(filePath) {
				loggedPassword = true
				postBodyStr := strings.Join(postBodySlice, "\n")
				utils.LogMessageToPath(
					"Found potential password in the post:\n\n"+postBodyStr,
					filePath,
				)
			}
		}

		utils.DetectOtherExtDLLink(text, postFolderPath)
		if utils.DetectGDriveLinks(text, postFolderPath, false) && downloadGdrive {
			detectedGdriveLinks = append(detectedGdriveLinks, map[string]string{
				"url":      text,
				"filepath": filepath.Join(postFolderPath, api.GdriveFolder),
			})
		}
	}
	return detectedGdriveLinks
}

// Pixiv Fanbox permitted file extensions based on
// https://fanbox.pixiv.help/hc/en-us/articles/360011057793-What-types-of-attachments-can-I-post-
var pixivFanboxAllowedImageExt = []string{"jpg", "jpeg", "png", "gif"}

// Process the JSON response from Pixiv Fanbox's API and
// returns a map of urls and a map of GDrive urls to download from
func ProcessFanboxPost(res *http.Response, postJsonArg interface{}, downloadPath string, pixivFanboxDlOptions *PixivFanboxDlOptions) ([]map[string]string, []map[string]string, error) {
	var err error
	var post interface{}
	var postJsonBody []byte
	if postJsonArg == nil {
		post, postJsonBody, err = utils.LoadJsonFromResponse(res)
		if err != nil {
			return nil, nil, err
		}
		if post == nil {
			return nil, nil, nil
		}
	} else {
		post = postJsonArg
	}

	postJson := post.(map[string]interface{})["body"].(map[string]interface{})
	postId := postJson["id"].(string)
	postTitle := postJson["title"].(string)
	creatorId := postJson["creatorId"].(string)
	postFolderPath := utils.GetPostFolder(filepath.Join(downloadPath, "Pixiv-Fanbox"), creatorId, postId, postTitle)

	var urlsMap []map[string]string
	thumbnail := postJson["coverImageUrl"]
	if pixivFanboxDlOptions.DlThumbnails && thumbnail != nil {
		urlsMap = append(urlsMap, map[string]string{
			"url":      thumbnail.(string),
			"filepath": postFolderPath,
		})
	}

	// Note that Pixiv Fanbox posts have 3 types of formatting (as of now):
	//	1. With proper formatting and mapping of post content elements ("article")
	//	2. With a simple formatting that obly contains info about the text and files ("file", "image")
	postType := postJson["type"].(string)
	postBody := postJson["body"]
	if postBody == nil {
		return urlsMap, nil, nil
	}
	postContent := postBody.(map[string]interface{})
	if postContent == nil {
		return urlsMap, nil, nil
	}

	var gdriveLinks []map[string]string
	switch postType {
	case "file", "image":
		// process the text in the post
		postBody := postContent["text"]
		detectedGdriveLinks := ProcessPixivFanboxText(postBody, postFolderPath, pixivFanboxDlOptions.DlGdrive)
		if detectedGdriveLinks != nil {
			gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
		}

		// retrieve images and attachments url(s)
		imageAndAttachmentUrls := postContent[postType+"s"]
		if (imageAndAttachmentUrls != nil) && (pixivFanboxDlOptions.DlImages || pixivFanboxDlOptions.DlAttachments) {
			for _, fileInfo := range imageAndAttachmentUrls.([]interface{}) {
				fileInfoMap := fileInfo.(map[string]interface{})

				var fileUrl, filename, extension string
				if postType == "image" {
					fileUrl = fileInfoMap["originalUrl"].(string)
					extension = fileInfoMap["extension"].(string)
					filename = utils.GetLastPartOfURL(fileUrl)
				} else { // postType == "file"
					fileUrl = fileInfoMap["url"].(string)
					extension = fileInfoMap["extension"].(string)
					filename = fileInfoMap["name"].(string) + "." + extension
				}

				var filePath string
				isImage := utils.SliceContains(pixivFanboxAllowedImageExt, extension)
				if isImage {
					filePath = filepath.Join(postFolderPath, api.ImagesFolder, filename)
				} else {
					filePath = filepath.Join(postFolderPath, api.AttachmentFolder, filename)
				}

				if (isImage && pixivFanboxDlOptions.DlImages) || (!isImage && pixivFanboxDlOptions.DlAttachments) {
					urlsMap = append(urlsMap, map[string]string{
						"url":      fileUrl,
						"filepath": filePath,
					})
				}
			}
		}
	case "article":
		// process the text in the post
		articleContents := postContent["blocks"]
		if articleContents != nil {
			articleContentsSlice := articleContents.([]interface{})
			loggedPassword := false
			for _, articleBlock := range articleContentsSlice {
				text := articleBlock.(map[string]interface{})["text"]
				if text != nil {
					textStr := text.(string)
					if utils.DetectPasswordInText(textStr) && !loggedPassword {
						// Log the entire post text if it contains a password
						filePath := filepath.Join(postFolderPath, api.PasswordFilename)
						if !utils.PathExists(filePath) {
							loggedPassword = true
							postBodyStr := "Found potential password in the post:\n\n"
							for _, articleContent := range articleContentsSlice {
								articleText := articleContent.(map[string]interface{})["text"]
								if articleText != nil {
									postBodyStr += articleText.(string) + "\n"
								}
							}
							utils.LogMessageToPath(
								postBodyStr,
								filePath,
							)
						}
					}

					utils.DetectOtherExtDLLink(textStr, postFolderPath)
					if utils.DetectGDriveLinks(textStr, postFolderPath, false) && pixivFanboxDlOptions.DlGdrive {
						gdriveLinks = append(gdriveLinks, map[string]string{
							"url":      textStr,
							"filepath": filepath.Join(postFolderPath, api.GdriveFolder),
						})
					}
				}

				articleLinks := articleBlock.(map[string]interface{})["links"]
				if articleLinks != nil {
					for _, link := range articleLinks.([]interface{}) {
						linkUrl := link.(map[string]interface{})["url"].(string)
						utils.DetectOtherExtDLLink(linkUrl, postFolderPath)
						if utils.DetectGDriveLinks(linkUrl, postFolderPath, true) && pixivFanboxDlOptions.DlGdrive {
							gdriveLinks = append(gdriveLinks, map[string]string{
								"url":      linkUrl,
								"filepath": filepath.Join(postFolderPath, api.GdriveFolder),
							})
							continue
						}
					}
				}
			}
		}

		// retrieve images and attachments url(s)
		images := postContent["imageMap"]
		if images != nil && pixivFanboxDlOptions.DlImages {
			imageMap := images.(map[string]interface{})
			for _, imageInfo := range imageMap {
				imageUrl := imageInfo.(map[string]interface{})["originalUrl"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, api.ImagesFolder),
				})
			}
		}

		attachments := postContent["fileMap"]
		if attachments != nil && pixivFanboxDlOptions.DlAttachments {
			attachmentMap := attachments.(map[string]interface{})
			for _, attachmentInfo := range attachmentMap {
				attachmentInfoMap := attachmentInfo.(map[string]interface{})
				attachmentUrl := attachmentInfoMap["url"].(string)
				filename := attachmentInfoMap["name"].(string) + "." + attachmentInfoMap["extension"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url":      attachmentUrl,
					"filepath": filepath.Join(postFolderPath, api.AttachmentFolder, filename),
				})
			}
		}
	case "text": // text post
		// Usually has no content but try to detect for any external download links
		detectedGdriveLinks := ProcessPixivFanboxText(
			postContent["text"],
			postFolderPath,
			pixivFanboxDlOptions.DlGdrive,
		)
		if detectedGdriveLinks != nil {
			gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
		}
	default: // unknown post type
		return nil, nil, fmt.Errorf(
			"pixiv fanbox error %d: unknown post type, \"%s\"\nPixiv Fanbox post content:\n%s",
			utils.JSON_ERROR,
			postType,
			string(postJsonBody),
		)
	}
	return urlsMap, gdriveLinks, nil
}

// Query Pixiv Fanbox's API based on the slice of post IDs and returns a map of
// urls and a map of GDrive urls to download from
func GetPostDetails(postIds []string, pixivFanboxDlOptions *PixivFanboxDlOptions) ([]map[string]string, []map[string]string) {
	maxConcurrency := utils.MAX_API_CALLS
	if len(postIds) < maxConcurrency {
		maxConcurrency = len(postIds)
	}
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(postIds))

	bar := utils.GetProgressBar(
		len(postIds),
		"Getting Pixiv Fanbox post details...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting %d Pixiv Fanbox post details!", len(postIds))),
	)
	url := "https://api.fanbox.cc/post.info"
	for _, postId := range postIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(postId string) {
			defer wg.Done()

			header := GetPixivFanboxHeaders()
			params := map[string]string{"postId": postId}
			res, err := request.CallRequest(
				"GET",
				url,
				30,
				pixivFanboxDlOptions.SessionCookies,
				header,
				params,
				false,
			)
			if err != nil || res.StatusCode != 200 {
				utils.LogError(err, fmt.Sprintf("failed to get post details for %s", url), false)
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

	var errSlice []error
	var urlsMap, gdriveUrls []map[string]string
	for res := range resChan {
		postUrls, postGdriveLinks, err := ProcessFanboxPost(
			res,
			nil,
			utils.DOWNLOAD_PATH,
			pixivFanboxDlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			urlsMap = append(urlsMap, postUrls...)
			gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		}
		bar.Add(1)
	}

	utils.LogErrors(false, nil, errSlice...)
	return urlsMap, gdriveUrls
}

type CreatorPaginatedPosts struct {
	Body []string `json:"body"`
}

type FanboxCreatorPosts struct {
	Body struct {
		Items []struct {
			Id string `json:"id"`
		} `json:"items"`
	} `json:"body"`
}

// GetFanboxCreatorPosts returns a slice of post IDs for a given creator
func GetFanboxPosts(creatorId string, cookies []http.Cookie) ([]string, error) {
	params := map[string]string{"creatorId": creatorId}
	headers := GetPixivFanboxHeaders()
	res, err := request.CallRequest(
		"GET",
		"https://api.fanbox.cc/post.paginateCreator",
		30,
		cookies,
		headers,
		params,
		false,
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

	var resJson CreatorPaginatedPosts
	resBody, err := utils.ReadResBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resBody, &resJson)
	if err != nil {
		err = fmt.Errorf(
			"pixiv fanbox error %d: failed to unmarshal json for Pixiv Fanbox creator's pages for %s\nJSON: %s",
			utils.JSON_ERROR,
			creatorId,
			string(resBody),
		)
		return nil, err
	}
	paginatedUrls := resJson.Body

	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if len(paginatedUrls) < maxConcurrency {
		maxConcurrency = len(paginatedUrls)
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(paginatedUrls))
	for _, paginatedUrl := range paginatedUrls {
		wg.Add(1)
		queue <- struct{}{}
		go func(reqUrl string) {
			defer wg.Done()
			res, err := request.CallRequest("GET", reqUrl, 30, cookies, headers, nil, false)
			if err != nil || res.StatusCode != 200 {
				if err == nil {
					res.Body.Close()
				}
				utils.LogError(err, fmt.Sprintf("failed to get post for %s", reqUrl), false)
			} else {
				resChan <- res
			}
			<-queue
		}(paginatedUrl)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	// parse the JSON response
	var errSlice []error
	var postIds []string
	for res := range resChan {
		resBody, err := utils.ReadResBody(res)
		if err != nil {
			errSlice = append(errSlice, err)
			continue
		}

		var resJson FanboxCreatorPosts
		err = json.Unmarshal(resBody, &resJson)
		if err != nil {
			err = fmt.Errorf(
				"pixiv fanbox error %d: failed to unmarshal json for Pixiv Fanbox creator's post\nJSON: %s",
				utils.JSON_ERROR,
				string(resBody),
			)
			errSlice = append(errSlice, err)
			continue
		}

		for _, postInfoMap := range resJson.Body.Items {
			postIds = append(postIds, postInfoMap.Id)
		}
	}

	utils.LogErrors(false, nil, errSlice...)
	return postIds, nil
}

// Retrieves all the posts based on the slice of creator IDs and returns a slice of post IDs
func GetCreatorsPosts(creatorIds []string, cookies []http.Cookie) []string {
	var postIds []string
	bar := utils.GetProgressBar(
		len(creatorIds),
		"Getting post IDs from creator(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting post IDs from %d creator(s)!", len(creatorIds))),
	)

	var errSlice []error
	for _, creatorId := range creatorIds {
		retrievedPostIds, err := GetFanboxPosts(creatorId, cookies)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			postIds = append(postIds, retrievedPostIds...)
		}

		bar.Add(1)
	}

	utils.LogErrors(false, nil, errSlice...)
	return postIds
}
