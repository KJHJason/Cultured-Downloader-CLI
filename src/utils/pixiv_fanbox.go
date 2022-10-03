package utils

import (
	"fmt"
	"sync"
	"strings"
	"net/http"
	"path/filepath"
	"github.com/panjf2000/ants/v2"
)

func ProcessFanboxPost(res *http.Response, postJsonArg interface{}, downloadPath string) ([]map[string]string, []map[string]string) {
	var post interface{}
	if postJsonArg == nil {
		post = LoadJsonFromResponse(res)
		if post == nil {
			return []map[string]string{}, []map[string]string{}
		}
	} else {
		post = postJsonArg
	}

	postJson := post.(map[string]interface{})["body"].(map[string]interface{})
	postId := postJson["id"].(string)
	postTitle := postJson["title"].(string)
	creatorId := postJson["creatorId"].(string)
	postFolderPath := CreatePostFolder(filepath.Join(downloadPath, "Pixiv-Fanbox"), creatorId, postId, postTitle)

	var urlsMap []map[string]string
	thumbnail := postJson["coverImageUrl"]
	if thumbnail != nil {
		urlsMap = append(urlsMap, map[string]string{
			"url": thumbnail.(string),
			"filepath": filepath.Join(postFolderPath),
		})
	}

	// Note that Pixiv Fanbox posts have 3 types of formatting (as of now):
	//	1. With proper formatting and mapping of post content elements ("article")
	//	2. With a simple formatting that obly contains info about the text and files ("file", "image")
	postType := postJson["type"].(string)
	postContent := postJson["body"].(map[string]interface{})
	if postContent == nil {
		return urlsMap, []map[string]string{}
	}

	var gdriveLinks []map[string]string
	switch postType {
	case "file", "image":
		// process the text in the post
		postBody := postContent["text"]
		if postBody != nil {
			postBodyArr := strings.FieldsFunc(
				postBody.(string),
				func(c rune) bool {return c == '\n'},
			)
			for idx, text := range postBodyArr {
				if DetectPasswordInText(postFolderPath, text) {
					// log the next element in the post body as a possible password
					if idx + 1 < len(postBodyArr) {
						nextText := postBodyArr[idx + 1]
						extraBlock := fmt.Sprintf(
							"Note: If the password was not present in the text above,\n" +
							"it might be in the next block of text:\n%s\n\n",
							nextText,
						)
						LogMessageToPath(
							extraBlock, 
							filepath.Join(postFolderPath, "detected_passwords.txt"),
						)
					}
				}
	
				DetectOtherExtDLLink(text, postFolderPath)
				if DetectGDriveLinks(text, postFolderPath, false) {
					gdriveLinks = append(gdriveLinks, map[string]string{
						"url": text,
						"filepath": postFolderPath,
					})
				}
			}
		}

		// retrieve images and attachments url(s)
		imageAndAttachmentUrls := postContent[postType + "s"]
		if imageAndAttachmentUrls != nil {
			for _, fileInfo := range imageAndAttachmentUrls.([]interface{}) {
				fileInfoMap := fileInfo.(map[string]interface{})
				fileUrl := fileInfoMap["originalUrl"]
				if fileUrl == nil {
					fileUrl = fileInfoMap["url"]
				}
				if fileUrl == nil {
					continue
				}

				urlsMap = append(urlsMap, map[string]string{
					"url": fileUrl.(string),
					"filepath": postFolderPath,
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
					if DetectGDriveLinks(textStr, postFolderPath, false) {
						gdriveLinks = append(gdriveLinks, map[string]string{
							"url": textStr,
							"filepath": postFolderPath,
						})
					}

					DetectOtherExtDLLink(textStr, postFolderPath)
					if DetectPasswordInText(postFolderPath, textStr) {
						// log the next two elements in the post body as a possible password
						extraBlocks := "Note: If the password was not present in the text above,\n" +
										"it might be in the next block of text:\n"
						for i := 1; i <= 2; i++ {
							if idx + i < len(articleContentsArr) {
								nextText := articleContentsArr[idx + i].(map[string]interface{})["text"]
								if nextText != nil {
									extraBlocks += nextText.(string) + "\n"
								}
							}
						}
						extraBlocks += "\n"
						LogMessageToPath(
							extraBlocks,
							filepath.Join(postFolderPath, "detected_passwords.txt"),
						)
					}
				}
				articleLinks := articleBlock.(map[string]interface{})["links"]
				if articleLinks != nil {
					for _, link := range articleLinks.([]interface{}) {
						linkUrl := link.(map[string]interface{})["url"].(string)
						DetectOtherExtDLLink(linkUrl, postFolderPath)
						if DetectGDriveLinks(linkUrl, postFolderPath, true) {
							gdriveLinks = append(gdriveLinks, map[string]string{
								"url": linkUrl,
								"filepath": postFolderPath,
							})
							continue
						}
					}
				}
			}
		}
		// retrieve images and attachments url(s)
		images := postContent["images"]
		if images != nil {
			imageMap := images.(map[string]interface{})
			for _, imageInfo := range imageMap {
				imageUrl := imageInfo.(map[string]interface{})["originalUrl"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url": imageUrl,
					"filepath": postFolderPath,
				})
			}
		}
		attachments := postContent["fileMap"]
		if attachments != nil {
			attachmentMap := attachments.(map[string]interface{})
			for _, attachmentInfo := range attachmentMap {
				attachmentUrl := attachmentInfo.(map[string]interface{})["url"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url": attachmentUrl,
					"filepath": postFolderPath,
				})
			}
		}
	default: // unknown post type
		panic(fmt.Sprintf("Unknown post type: %s\nPlease report it as a bug!", postType))
	}
	return urlsMap, gdriveLinks
}

func GetFanboxPosts(creatorId string, cookies []http.Cookie) []map[string]string {
	params := map[string]string{"creatorId": creatorId}
	headers := map[string]string{"Origin": "https://www.fanbox.cc", "Referer": "https://www.fanbox.cc/"}
	res, err := CallRequest("GET", GetAPICreatorPages("fanbox", creatorId), 30, cookies, headers, params, false)
	if err != nil || res.StatusCode != 200 {
		LogError(err, fmt.Sprintf("failed to get creator's pages for %s", creatorId), false)
		return []map[string]string {}
	}

	// parse the response
	paginatedUrls := []string{}
	resJson := LoadJsonFromResponse(res)
	posts := resJson.(map[string]interface{})["body"]
	if posts == nil {
		return []map[string]string {}
	}
	for _, post := range posts.([]interface{}) {
		paginatedUrls = append(paginatedUrls, post.(string))
	}

	var wg sync.WaitGroup
	maxConcurrency := MAX_CONCURRENT_DOWNLOADS
	if len(paginatedUrls) < MAX_CONCURRENT_DOWNLOADS {
		maxConcurrency = len(paginatedUrls)
	}
	pool, _ := ants.NewPool(maxConcurrency)
	defer pool.Release()
	resChan := make(chan *http.Response, len(paginatedUrls))
	for _,  url := range paginatedUrls {
		wg.Add(1)
		err = pool.Submit(func() {
			defer wg.Done()
			res, err := CallRequest("GET", url, 30, cookies, headers, params, false)
			if err != nil || res.StatusCode != 200 {
				LogError(err, fmt.Sprintf("failed to get post for %s", url), false)
			} else {
				resChan <- res
			}
		})
		if err != nil {
			panic(err)
		}
	}
	wg.Wait()
	close(resChan)

	// parse the JSON response
	var postIds []map[string]string
	for res := range resChan {
		resJson := LoadJsonFromResponse(res)
		if resJson == nil {
			continue
		}
		postInfoArr := resJson.(map[string]interface{})["body"].(map[string]interface{})["items"]
		if postInfoArr == nil {
			continue
		}
		for _, postInfo := range postInfoArr.([]interface{}) {
			postInfoMap := postInfo.(map[string]interface{})
			postId := postInfoMap["id"].(string)
			creatorId := postInfoMap["creatorId"].(string)
			postIds = append(postIds, map[string]string{"postId": postId, "creatorId": creatorId})
		}
	}
	return postIds
}