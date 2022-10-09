package pixiv_fanbox

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"encoding/json"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

var (
	// Pixiv Fanbox permitted file extensions based on
	//	https://fanbox.pixiv.help/hc/en-us/articles/360011057793-What-types-of-attachments-can-I-post-
	pixivFanboxAllowedImageExt = []string{"jpg", "jpeg", "png", "gif"}
)

// Returns a defined request header needed to communicate with Pixiv Fanbox's API
func GetPixivFanboxHeaders() map[string]string {
	return map[string]string{
		"Origin": "https://www.fanbox.cc", 
		"Referer": "https://www.fanbox.cc/",
	}
}

// Process the JSON response from Pixiv Fanbox's API and 
// returns a map of urls and a map of GDrive urls to download from
func ProcessFanboxPost(
	res *http.Response, postJsonArg interface{}, downloadPath string, 
	downloadThumbnail, downloadImages, downloadAttachments, downloadGdrive bool,
) ([]map[string]string, []map[string]string) {
	var post interface{}
	if postJsonArg == nil {
		post = utils.LoadJsonFromResponse(res)
		if post == nil {
			return nil, nil
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
	if downloadThumbnail && thumbnail != nil {
		urlsMap = append(urlsMap, map[string]string{
			"url":      thumbnail.(string),
			"filepath": postFolderPath,
		})
	}

	// Note that Pixiv Fanbox posts have 3 types of formatting (as of now):
	//	1. With proper formatting and mapping of post content elements ("article")
	//	2. With a simple formatting that obly contains info about the text and files ("file", "image")
	postType := postJson["type"].(string)
	postContent := postJson["body"].(map[string]interface{})
	if postContent == nil {
		return urlsMap, nil
	}

	var gdriveLinks []map[string]string
	switch postType {
	case "file", "image":
		// process the text in the post
		postBody := postContent["text"]
		if postBody != nil {
			postBodyArr := strings.FieldsFunc(
				postBody.(string),
				func(c rune) bool { return c == '\n' },
			)
			loggedPassword := false
			for _, text := range postBodyArr {
				if utils.DetectPasswordInText(text) && !loggedPassword {
					// Log the entire post text if it contains a password
					filePath := filepath.Join(postFolderPath, api.PasswordFilename)
					if !utils.PathExists(filePath) {
						loggedPassword = true
						postBodyStr := strings.Join(postBodyArr, "\n")
						utils.LogMessageToPath(
							"Found potential password in the post:\n\n" + postBodyStr,
							filePath,
						)
					}
				}

				utils.DetectOtherExtDLLink(text, postFolderPath)
				if utils.DetectGDriveLinks(text, postFolderPath, false) && downloadGdrive {
					gdriveLinks = append(gdriveLinks, map[string]string{
						"url":      text,
						"filepath": filepath.Join(postFolderPath, api.GdriveFolder),
					})
				}
			}
		}

		// retrieve images and attachments url(s)
		imageAndAttachmentUrls := postContent[postType+"s"]
		if (imageAndAttachmentUrls != nil) && (downloadImages || downloadAttachments) {
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
				isImage := utils.ArrContains(pixivFanboxAllowedImageExt, extension)
				if isImage {
					filePath = filepath.Join(postFolderPath, api.ImagesFolder, filename)
				} else {
					filePath = filepath.Join(postFolderPath, api.AttachmentFolder, filename)
				}

				if (isImage && downloadImages) || (!isImage && downloadAttachments) {
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
			articleContentsArr := articleContents.([]interface{})
			loggedPassword := false
			for _, articleBlock := range articleContentsArr {
				text := articleBlock.(map[string]interface{})["text"]
				if text != nil {
					textStr := text.(string)
					if utils.DetectPasswordInText(textStr) && !loggedPassword {
						// Log the entire post text if it contains a password
						filePath := filepath.Join(postFolderPath, api.PasswordFilename)
						if !utils.PathExists(filePath) {
							loggedPassword = true
							postBodyStr := "Found potential password in the post:\n\n"
							for _, articleContent := range articleContentsArr {
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
					if utils.DetectGDriveLinks(textStr, postFolderPath, false) && downloadGdrive {
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
						if utils.DetectGDriveLinks(linkUrl, postFolderPath, true) && downloadGdrive {
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
		if images != nil && downloadImages {
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
		if attachments != nil && downloadAttachments {
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
	default: // unknown post type
		panic(fmt.Sprintf("Unknown post type: %s\nPlease report it as a bug!", postType))
	}
	return urlsMap, gdriveLinks
}

// Query Pixiv Fanbox's API based on the slice of post IDs and returns a map of 
// urls and a map of GDrive urls to download from
func GetPostDetails(
	postIds []string, cookies []http.Cookie,
	downloadThumbnail, downloadImages, downloadAttachments, downloadGdrive bool,
) ([]map[string]string, []map[string]string) {
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
			res, err := request.CallRequest("GET", url, 30, cookies, header, params, false)
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
	var urlsMap, gdriveUrls []map[string]string
	for res := range resChan {
		postUrls, postGdriveLinks := ProcessFanboxPost(
			res, nil, utils.DOWNLOAD_PATH, 
			downloadThumbnail, downloadImages, downloadAttachments, downloadGdrive,
		)
		urlsMap = append(urlsMap, postUrls...)
		gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		bar.Add(1)
	}
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
func GetFanboxPosts(creatorId string, cookies []http.Cookie) []string {
	params := map[string]string{"creatorId": creatorId}
	headers := GetPixivFanboxHeaders()
	res, err := request.CallRequest("GET", "https://api.fanbox.cc/post.paginateCreator", 30, cookies, headers, params, false)
	if err != nil || res.StatusCode != 200 {
		if err == nil {
			res.Body.Close()
		}
		utils.LogError(err, fmt.Sprintf("failed to get creator's pages for %s", creatorId), false)
		return nil
	}

	var resJson CreatorPaginatedPosts
	resBody := utils.ReadResBody(res)
	err = json.Unmarshal(resBody, &resJson)
	if err != nil {
		utils.LogError(err, fmt.Sprintf("Failed to unmarshal json for Pixiv Fanbox creator's pages\nJSON: %s", string(resBody)), false)
		return nil
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
	var postIds []string
	for res := range resChan {
		resBody := utils.ReadResBody(res)
		var resJson FanboxCreatorPosts
		err = json.Unmarshal(resBody, &resJson)
		if err != nil {
			utils.LogError(err, fmt.Sprintf("Failed to unmarshal json for Pixiv Fanbox creator's post\nJSON: %s", string(resBody)), false)
			continue
		}

		for _, postInfoMap := range resJson.Body.Items {
			postIds = append(postIds, postInfoMap.Id)
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
	for _, creatorId := range creatorIds {
		postIds = append(postIds, GetFanboxPosts(creatorId, cookies)...)
		bar.Add(1)
	}
	return postIds
}