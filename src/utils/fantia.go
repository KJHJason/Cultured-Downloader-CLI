package utils

import (
	"fmt"
	"net/http"
	"path/filepath"
	"github.com/PuerkitoBio/goquery"
)

func GetFantiaPosts(creatorId string, cookies []http.Cookie) []string {
	var postIds []string
	pageNum := 1
	for {
		url := GetAPICreatorPages(Fantia, creatorId)
		params := map[string]string{
			"page":   fmt.Sprintf("%d", pageNum),
			"q[s]":   "newer",
			"q[tag]": "",
		}
		res, err := CallRequest("GET", url, 30, cookies, nil, params, false)
		if err != nil || res.StatusCode != 200 {
			if err == nil {
				res.Body.Close()
			}
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
			if href, exists := s.Attr("href"); exists {
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

func ProcessFantiaPost(res *http.Response, downloadPath string) []map[string]string {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	post := LoadJsonFromResponse(res)
	if post == nil {
		return nil
	}

	postJson := post.(map[string]interface{})["post"].(map[string]interface{})
	postId := fmt.Sprintf("%d", int64(postJson["id"].(float64)))
	postTitle := postJson["title"].(string)
	creatorName := postJson["fanclub"].(map[string]interface{})["user"].(map[string]interface{})["name"].(string)
	postFolderPath := CreatePostFolder(filepath.Join(downloadPath, FantiaTitle), creatorName, postId, postTitle)

	var urlsMap []map[string]string
	thumbnail := postJson["thumb"].(map[string]interface{})
	if thumbnail != nil {
		thumbnailUrl := thumbnail["original"].(string)
		urlsMap = append(urlsMap, map[string]string{
			"url":      thumbnailUrl,
			"filepath": postFolderPath,
		})
	}

	// get an array of maps in ["post_contents"]
	postContent := postJson["post_contents"]
	if postContent == nil {
		return urlsMap
	}
	for _, content := range postContent.([]interface{}) {
		// get post_content_photos if exists
		contentMap := content.(map[string]interface{})
		postContentPhotos := contentMap["post_content_photos"]
		if postContentPhotos != nil {
			// get an array of maps in ["post_content_photos"]
			images := postContentPhotos.([]interface{})
			for _, image := range images {
				// image url via ["url"]["original"]
				imageUrl := image.(map[string]interface{})["url"].(map[string]interface{})["original"].(string)
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, imagesFolder),
				})
			}
		}

		// get the attachment url string if it exists
		attachmentUrl := contentMap["attachment_url"]
		if attachmentUrl != nil {
			attachmentUrlStr := "https://fantia.jp" + attachmentUrl.(string)
			urlsMap = append(urlsMap, map[string]string{
				"url":      attachmentUrlStr,
				"filepath": filepath.Join(postFolderPath, attachmentFolder),
			})
		}
	}
	return urlsMap
}
