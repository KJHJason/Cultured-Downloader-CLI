package api

import (
	"fmt"
	"net/http"
	"sync"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

// Returns the corresponding API link 
// based on the given website string to get the post details
func GetAPIPostLink(website, postId string) string {
	if website == Fantia {
		return "https://fantia.jp/api/v1/posts/" + postId
	} else if website == PixivFanbox {
		return "https://api.fanbox.cc/post.info"
	} else {
		panic("invalid website")
	}
}

// Returns the corresponding API link
// based on the given website string to get the creator's posts
func GetAPICreatorPages(website, creatorId string) string {
	if website == Fantia {
		return "https://fantia.jp/fanclubs/" + creatorId + "/posts"
	} else if website == PixivFanbox {
		return "https://api.fanbox.cc/post.paginateCreator"
	} else {
		panic("invalid website")
	}
}

// Query the respective API based on the slice of post IDs and returns a map of 
// urls and a map of GDrive urls (For Pixiv Fanbox only) to download from
func GetPostDetails(postIds []string, website string, cookies []http.Cookie) ([]map[string]string, []map[string]string) {
	maxConcurrency := utils.MAX_API_CALLS
	if len(postIds) < maxConcurrency {
		maxConcurrency = len(postIds)
	}
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(postIds))

	bar := utils.GetProgressBar(
		len(postIds), 
		"Getting post details...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting %d post details!", len(postIds))),
	)
	for _, postId := range postIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(postId string) {
			defer wg.Done()
			url := GetAPIPostLink(website, postId)
			var header, params map[string]string
			if website == Fantia {
				header = map[string]string{"Referer": "https://fantia.jp/posts/" + postId}
			} else if website == PixivFanbox {
				header = GetPixivFanboxHeaders()
				params = map[string]string{"postId": postId}
			} else {
				panic("invalid website")
			}

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
		if website == Fantia {
			urlsMap = append(urlsMap, ProcessFantiaPost(res, utils.DOWNLOAD_PATH)...)
		} else if website == PixivFanbox {
			postUrls, postGdriveLinks := ProcessFanboxPost(res, nil, utils.DOWNLOAD_PATH)
			urlsMap = append(urlsMap, postUrls...)
			gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		} else {
			panic("invalid website")
		}
		bar.Add(1)
	}
	return urlsMap, gdriveUrls
}

// Retrieves all the posts based on the slice of creator IDs and returns a slice of post IDs
func GetCreatorsPosts(creatorIds []string, website string, cookies []http.Cookie) []string {
	var postIds []string
	bar := utils.GetProgressBar(
		len(creatorIds), 
		"Getting post IDs from creator(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting post IDs from %d creator(s)!", len(creatorIds))),
	)
	if website == Fantia {
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
	} else if website == PixivFanbox {
		for _, creatorId := range creatorIds {
			postIds = append(postIds, GetFanboxPosts(creatorId, cookies)...)
			bar.Add(1)
		}
	} else {
		panic("invalid website")
	}
	return postIds
}