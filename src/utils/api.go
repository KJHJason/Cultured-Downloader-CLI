package utils

import (
	"io"
	"fmt"
	"sync"
	"strings"
	"net/http"
	"path/filepath"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
)

func GetAPIPostLink(website string, postId string) string {
	if website == "fantia" {
		return "https://fantia.jp/api/v1/posts/" + postId
	} else if website == "fanbox" {
		return "https://api.fanbox.cc/post.info"
	} else {
		panic("invalid website")
	}
}

func GetAPICreatorPages(website string, creatorId string) string {
	if website == "fantia" {
		return "https://fantia.jp/fanclubs/" + creatorId + "/posts"
	} else if website == "fanbox" {
		return "https://api.fanbox.cc/post.paginateCreator"
	} else {
		panic("invalid website")
	}
}

func GetFantiaPosts(creatorId string, cookies []http.Cookie) []string {
	var postIds []string
	pageNum := 1
	for {
		url := GetAPICreatorPages("fantia", creatorId)
		params := map[string]string{
			"page": fmt.Sprintf("%d", pageNum),
			"q[s]": "newer",
			"q[tag]": "",
		}
		res, err := CallRequest(url, 30, cookies, "GET", nil, params)
		if err != nil {
			res.Body.Close()
			LogError(err, fmt.Sprintf("failed to get creator's pages for %s", url), false)
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
			fmt.Println(s.Attr("href"))
			href, exists := s.Attr("href")
			if exists {
				postIds = append(postIds, href)
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

func GetFanboxPosts(creatorId string, cookies []http.Cookie) []map[string]string {
	params := map[string]string{"creatorId": creatorId}
	headers := map[string]string{"Origin": "https://www.fanbox.cc", "Referer": "https://www.fanbox.cc/"}
	res, err := CallRequest(GetAPICreatorPages("fanbox", creatorId), 30, cookies, "GET", headers, params)
	if err != nil {
		LogError(err, fmt.Sprintf("failed to get creator's pages for %s", creatorId), false)
		return []map[string]string {}
	}

	// parse the response
	paginatedUrls := []string{}
	resJson := LoadJsonFromResponse(*res)
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
	sem := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(paginatedUrls))
	for _,  url := range paginatedUrls {
		wg.Add(1)
		sem <- struct{}{}
		go func(url string) {
			defer wg.Done()
			res, err := CallRequest(url, 30, cookies, "GET", headers, params)
			if err != nil {
				LogError(err, fmt.Sprintf("failed to get post for %s", url), false)
			} else {
				resChan <- res
			}
			<-sem
		}(url)
	}
	close(sem)
	wg.Wait()
	close(resChan)

	// parse the JSON response
	var postIds []map[string]string
	for res := range resChan {
		resJson := LoadJsonFromResponse(*res)
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

func GetCreatorsPosts(creatorIds []string, website string, cookies []http.Cookie) ([]map[string]string, []string) {
	var fantiaPostIds []string
	var fanboxPostIds []map[string]string

	if website == "fantia" {
		var wg sync.WaitGroup
		maxConcurrency := MAX_CONCURRENT_DOWNLOADS
		if len(creatorIds) < MAX_CONCURRENT_DOWNLOADS {
			maxConcurrency = len(creatorIds)
		}
		sem := make(chan struct{}, maxConcurrency)
		resChan := make(chan []string, len(creatorIds))
		for _, creatorId := range creatorIds {
			wg.Add(1)
			sem <- struct{}{}
			go func(creatorId string) {
				defer wg.Done()
				resChan <- GetFantiaPosts(creatorId, cookies)
				<-sem
			}(creatorId)
		}
		close(sem)
		wg.Wait()
		close(resChan)

		for postIds := range resChan {
			fantiaPostIds = append(fantiaPostIds, postIds...)
		}
	} else if website == "fanbox" {
		for _, creatorId := range creatorIds {
			fanboxPostIds = append(fanboxPostIds, GetFanboxPosts(creatorId, cookies)...)
		}
	} else {
		panic("invalid website")
	}

	return fanboxPostIds, fantiaPostIds
}

func GetPostDetails(postIdsOrUrls []string, website string, cookies []http.Cookie) ([]map[string]string, []map[string]string) {
	var wg sync.WaitGroup

	maxConcurrency := MAX_CONCURRENT_DOWNLOADS
	if len(postIdsOrUrls) < MAX_CONCURRENT_DOWNLOADS {
		maxConcurrency = len(postIdsOrUrls)
	}
	sem := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(postIdsOrUrls))
	for _,  postIdOrUrl := range postIdsOrUrls {
		wg.Add(1)
		sem <- struct{}{}
		go func(postIdOrUrl string) {
			defer wg.Done()

			postId := GetLastPartOfURL(postIdOrUrl)
			url := GetAPIPostLink(website, postId)
			var header map[string]string
			var params map[string]string
			if website == "fantia" {
				header = map[string]string{"Referer": "https://fantia.jp/posts/" + postId}
			} else if website == "fanbox" {
				header = map[string]string{
					"Referer": postIdOrUrl,
					"Origin": "https://www.fanbox.cc",
				}
				params = map[string]string{"postId": postId}
			} else {
				panic("invalid website")
			}

			res, err := CallRequest(url, 30, cookies, "GET", header, params)
			if err != nil {
				LogError(err, fmt.Sprintf("failed to get post details for %s", url), false)
			} else {
				resChan <- res
			}
			<-sem
		}(postIdOrUrl)
	}
	close(sem)
	wg.Wait()
	close(resChan)

	// parse the responses
	var urlsMap []map[string]string
	var gdriveUrls []map[string]string
	for res := range resChan {
		if website == "fantia" {
			urlsMap = append(urlsMap, ProcessFantiaPost(*res, DOWNLOAD_PATH)...)
		} else if website == "fanbox" {
			postUrls, postGdriveLinks := ProcessFanboxPost(*res, nil, DOWNLOAD_PATH)
			urlsMap = append(urlsMap, postUrls...)
			gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		} else {
			panic("invalid website")
		}
	}
	return urlsMap, gdriveUrls
}

func DetectPasswordInText(postFolderPath string, text string) bool {
	passwordFilename := "detected_passwords.txt"
	passwordFilepath := filepath.Join(postFolderPath, passwordFilename)
	for _, passwordText := range PASSWORD_TEXTS {
		if strings.Contains(text, passwordText) {
			passwordText := fmt.Sprintf(
				"Detected a possible password-protected content in post: %s\n\n",
				text,
			)
			LogMessageToPath(passwordText, passwordFilepath)
			return true
		}
	}
	return false
}

func DetectGDriveLinks(text string, isUrl bool, postFolderPath string) bool {
	gdriveFilename := "detected_gdrive_links.txt"
	gdriveFilepath := filepath.Join(postFolderPath, gdriveFilename)
	driveSubstr := "https://drive.google.com"
	containsGDriveLink := false
	if isUrl && strings.HasPrefix(text, driveSubstr) {
		containsGDriveLink = true
	} else if strings.Contains(text, driveSubstr) {
		containsGDriveLink = true
	}

	if !containsGDriveLink {
		return false
	}

	gdriveText := fmt.Sprintf(
		"Google Drive link detected: %s\n\n",
		text,
	)
	LogMessageToPath(gdriveText, gdriveFilepath)
	return true
}

func DetectOtherExtDLLink(text string, postFolderPath string) bool {
	otherExtFilename := "detected_external_links.txt"
	otherExtFilepath := filepath.Join(postFolderPath, otherExtFilename)
	for _, extDownloadProvider := range EXTERNAL_DOWNLOAD_PLATFORMS {
		if strings.Contains(text, extDownloadProvider) {
			otherExtText := fmt.Sprintf(
				"Detected a link that points to an external file hosting in post's description:\n%s\n\n",
				text,
			)
			LogMessageToPath(otherExtText, otherExtFilepath)
			return true
		}
	}
	return false
}

func LoadJsonFromResponse(res http.Response) interface{} {
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var post interface{}
	err = json.Unmarshal(body, &post)
	if (err != nil) {
		errorMsg := fmt.Sprintf(
			"failed to parse json response from %s due to %v", 
			res.Request.URL.String(), 
			err,
		)
		LogError(err, errorMsg, false)
		return interface{}(nil)
	}
	return post
}

func ProcessFantiaPost(res http.Response, downloadPath string) []map[string]string {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	post := LoadJsonFromResponse(res)
	if post == nil {
		return []map[string]string{}
	}

	postJson := post.(map[string]interface{})["post"].(map[string]interface{})
	postId := fmt.Sprintf("%d",int64(postJson["id"].(float64)))
	postTitle := postJson["title"].(string)
	creatorName := postJson["fanclub"].(map[string]interface{})["user"].(map[string]interface{})["name"].(string)
	postFolderPath := CreatePostFolder(downloadPath, creatorName, postId, postTitle)

	var urlsMap []map[string]string
	thumbnail := postJson["thumb"].(map[string]interface{})
	if thumbnail != nil {
		thumbnailUrl := thumbnail["original"].(string)
		urlsMap = append(urlsMap, map[string]string{
			"url": thumbnailUrl,
			"filepath": filepath.Join(postFolderPath),
		})
	}

	// get an array of maps in ["post_contents"]
	postContent := postJson["post_contents"]
	if postContent == nil {
		return urlsMap
	}
	for _, content := range postContent.([]interface{}) {
		// get post_content_photos if exists
		postContentPhotos := content.(map[string]interface{})["post_content_photos"]
		if postContentPhotos != nil {
			// get an array of maps in ["post_content_photos"]
			images := postContentPhotos.([]interface{})
			for _, image := range images {
				// image url via ["url"]["original"]
				imageUrl := image.(map[string]interface{})["url"].(map[string]interface{})["original"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url": imageUrl,
					"filepath": filepath.Join(postFolderPath, "images"),
				})
			}
		}

		// get the attachment url string if it exists
		attachmentUrl := content.(map[string]interface{})["attachment_url"]
		if attachmentUrl != nil {
			attachmentUrlStr := "https://fantia.jp" + attachmentUrl.(string)
			urlsMap = append(urlsMap, map[string]string{
				"url": attachmentUrlStr,
				"filepath": filepath.Join(postFolderPath, "attachments"),
			})
		}
	}

	return urlsMap
}

func ProcessFanboxPost(res http.Response, postJsonArg interface{}, downloadPath string) ([]map[string]string, []map[string]string) {
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
	postFolderPath := CreatePostFolder(downloadPath, creatorId, postId, postTitle)

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
			postBodyArr := postBody.([]interface{})
			for idx, text := range postBodyArr {
				textStr := text.(string)
				if DetectPasswordInText(postFolderPath, textStr) {
					// log the next element in the post body as a possible password
					if idx + 1 < len(postBodyArr) {
						nextText := postBodyArr[idx + 1].(string)
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
	
				DetectOtherExtDLLink(textStr, postFolderPath)
				if DetectGDriveLinks(textStr, false, postFolderPath) {
					gdriveLinks = append(gdriveLinks, map[string]string{
						"url": textStr,
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
				fileUrl := fileInfoMap["originalUrl"].(string)
				if fileUrl == "" {
					fileUrl = fileInfoMap["url"].(string)
				}
				if fileUrl == "" {
					continue
				}

				urlsMap = append(urlsMap, map[string]string{
					"url": fileUrl,
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
					if DetectGDriveLinks(textStr, false, postFolderPath) {
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
						if DetectGDriveLinks(linkUrl, true, postFolderPath) {
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