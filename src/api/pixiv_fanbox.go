package api

import (
	"fmt"
	"net/url"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"encoding/json"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

var (
	// Pixiv Fanbox permitted file extensions based on
	//	https://fanbox.pixiv.help/hc/en-us/articles/360011057793-What-types-of-attachments-can-I-post-
	pixivFanboxAllowedImageExt = []string{"jpg", "jpeg", "png", "gif"}
)

func GetPixivFanboxHeaders() map[string]string {
	return map[string]string{
		"Origin": "https://www.fanbox.cc", 
		"Referer": "https://www.fanbox.cc/",
	}
}

func ProcessFanboxPost(res *http.Response, postJsonArg interface{}, downloadPath string) ([]map[string]string, []map[string]string) {
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
	postFolderPath := utils.CreatePostFolder(filepath.Join(downloadPath, "Pixiv-Fanbox"), creatorId, postId, postTitle)

	var urlsMap []map[string]string
	thumbnail := postJson["coverImageUrl"]
	if thumbnail != nil {
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
			for idx, text := range postBodyArr {
				if utils.DetectPasswordInText(postFolderPath, text) {
					// log the next element in the post body as a possible password
					if idx+1 < len(postBodyArr) {
						nextText := postBodyArr[idx+1]
						extraBlock := fmt.Sprintf(
							"Note: If the password was not present in the text above,\n"+
								"it might be in the next block of text:\n%s\n\n",
							nextText,
						)
						utils.LogMessageToPath(
							extraBlock,
							filepath.Join(postFolderPath, "detected_passwords.txt"),
						)
					}
				}

				utils.DetectOtherExtDLLink(text, postFolderPath)
				if utils.DetectGDriveLinks(text, postFolderPath, false) {
					gdriveLinks = append(gdriveLinks, map[string]string{
						"url":      text,
						"filepath": filepath.Join(gdriveFolder, postFolderPath),
					})
				}
			}
		}

		// retrieve images and attachments url(s)
		imageAndAttachmentUrls := postContent[postType+"s"]
		if imageAndAttachmentUrls != nil {
			for _, fileInfo := range imageAndAttachmentUrls.([]interface{}) {
				fileInfoMap := fileInfo.(map[string]interface{})

				var fileUrl, filename, extension string
				if postType == "file" {
					fileUrl = fileInfoMap["originalUrl"].(string)
					extension = fileInfoMap["extension"].(string)
					filename = fileInfoMap["name"].(string) + "." + extension
				} else {
					fileUrl = fileInfoMap["url"].(string)
					extension = fileInfoMap["extension"].(string)
					filename = utils.GetLastPartOfURL(fileUrl)
				}

				var filePath string
				if utils.ArrContains(pixivFanboxAllowedImageExt, extension) {
					filePath = filepath.Join(postFolderPath, imagesFolder, filename)
				} else {
					filePath = filepath.Join(postFolderPath, attachmentFolder, filename)
				}

				urlsMap = append(urlsMap, map[string]string{
					"url":      fileUrl,
					"filepath": filePath,
				})
			}
		}
	case "article":
		// process the text in the post
		articleContents := postContent["blocks"]
		if articleContents != nil {
			articleContentsArr := articleContents.([]interface{})
			for idx, articleBlock := range articleContentsArr {
				text := articleBlock.(map[string]interface{})["text"]
				if text != nil {
					textStr := text.(string)
					if utils.DetectGDriveLinks(textStr, postFolderPath, false) {
						gdriveLinks = append(gdriveLinks, map[string]string{
							"url":      textStr,
							"filepath": filepath.Join(postFolderPath, gdriveFolder),
						})
					}

					utils.DetectOtherExtDLLink(textStr, postFolderPath)
					if utils.DetectPasswordInText(postFolderPath, textStr) {
						// log the next two elements in the post body as a possible password
						extraBlocks := "Note: If the password was not present in the text above,\n" +
							"it might be in the next block of text:\n"
						for i := 1; i <= 2; i++ {
							if idx+i < len(articleContentsArr) {
								nextText := articleContentsArr[idx+i].(map[string]interface{})["text"]
								if nextText != nil {
									extraBlocks += nextText.(string) + "\n"
								}
							}
						}
						extraBlocks += "\n"
						utils.LogMessageToPath(
							extraBlocks,
							filepath.Join(postFolderPath, "detected_passwords.txt"),
						)
					}
				}
				articleLinks := articleBlock.(map[string]interface{})["links"]
				if articleLinks != nil {
					for _, link := range articleLinks.([]interface{}) {
						linkUrl := link.(map[string]interface{})["url"].(string)
						utils.DetectOtherExtDLLink(linkUrl, postFolderPath)
						if utils.DetectGDriveLinks(linkUrl, postFolderPath, true) {
							gdriveLinks = append(gdriveLinks, map[string]string{
								"url":      linkUrl,
								"filepath": filepath.Join(postFolderPath, gdriveFolder),
							})
							continue
						}
					}
				}
			}
		}
		// retrieve images and attachments url(s)
		images := postContent["imageMap"]
		if images != nil {
			imageMap := images.(map[string]interface{})
			for _, imageInfo := range imageMap {
				imageUrl := imageInfo.(map[string]interface{})["originalUrl"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, imagesFolder),
				})
			}
		}
		attachments := postContent["fileMap"]
		if attachments != nil {
			attachmentMap := attachments.(map[string]interface{})
			for _, attachmentInfo := range attachmentMap {
				attachmentInfoMap := attachmentInfo.(map[string]interface{})
				attachmentUrl := attachmentInfoMap["url"].(string)
				filename := attachmentInfoMap["name"].(string) + "." + attachmentInfoMap["extension"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url":      attachmentUrl,
					"filepath": filepath.Join(postFolderPath, attachmentFolder, filename),
				})
			}
		}
	default: // unknown post type
		panic(fmt.Sprintf("Unknown post type: %s\nPlease report it as a bug!", postType))
	}
	return urlsMap, gdriveLinks
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

func GetFanboxPosts(creatorId string, cookies []http.Cookie) []string {
	params := map[string]string{"creatorId": creatorId}
	headers := GetPixivFanboxHeaders()
	res, err := request.CallRequest("GET", GetAPICreatorPages(PixivFanbox, creatorId), 30, cookies, headers, params, false)
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
	paginationParamsArr := []map[string]string{}
	for _, paginatedPost := range resJson.Body {
		parsedUrl, _ := url.Parse(paginatedPost)
		parsedParamsMap := map[string]string{}
		for key, value := range parsedUrl.Query() {
			// since the Query will return a 
			// map of slices of strings
			parsedParamsMap[key] = value[0]
		}
		paginationParamsArr = append(paginationParamsArr, parsedParamsMap)
	}

	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if len(paginationParamsArr) < maxConcurrency {
		maxConcurrency = len(paginationParamsArr)
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(paginationParamsArr))
	for _, paginationParams := range paginationParamsArr {
		wg.Add(1)
		queue <- struct{}{}
		go func(params map[string]string) {
			defer wg.Done()
			res, err := request.CallRequest("GET", "https://api.fanbox.cc/post.listCreator", 30, cookies, headers, params, false)
			if err != nil || res.StatusCode != 200 {
				if err == nil {
					res.Body.Close()
				}
				url := "https://api.fanbox.cc/post.listCreator" + utils.ParamsToString(params)
				utils.LogError(err, fmt.Sprintf("failed to get post for %s", url), false)
			} else {
				resChan <- res
			}
			<-queue
		}(paginationParams)
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