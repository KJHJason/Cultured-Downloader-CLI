package api

import (
	"fmt"
	"strconv"
	"net/http"
	"encoding/json"
	"path/filepath"
	"github.com/PuerkitoBio/goquery"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

// Get all the creator's posts by using goquery to parse the HTML response to get the post IDs
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
			AttachmentURI string `json:"attachment_uri"`
			PostContentPhotos []struct {
				ID int `json:"id"`
				URL struct {
					Original string `json:"original"`
				} `json:"url"`
			} `json:"post_content_photos"`
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
	postFolderPath := utils.CreatePostFolder(filepath.Join(downloadPath, FantiaTitle), creatorName, postId, postTitle)

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
			postContentPhotos := content.PostContentPhotos
			for _, image := range postContentPhotos {
				imageUrl := image.URL.Original
				urlsMap = append(urlsMap, map[string]string{
					"url":      imageUrl,
					"filepath": filepath.Join(postFolderPath, imagesFolder),
				})
			}
		}

		if downloadAttachments {
			// get the attachment url string if it exists
			attachmentUrl := content.AttachmentURI
			if attachmentUrl != "" {
				attachmentUrlStr := "https://fantia.jp" + attachmentUrl
				urlsMap = append(urlsMap, map[string]string{
					"url":      attachmentUrlStr,
					"filepath": filepath.Join(postFolderPath, attachmentFolder),
				})
			}
		}
	}
	return urlsMap
}
