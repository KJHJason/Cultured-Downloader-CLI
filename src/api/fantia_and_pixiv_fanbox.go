package api

import (
	"fmt"
	"net/http"
	"sync"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

func GetAPIPostLink(website, postId string) string {
	if website == Fantia {
		return "https://fantia.jp/api/v1/posts/" + postId
	} else if website == PixivFanbox {
		return "https://api.fanbox.cc/post.info"
	} else {
		panic("invalid website")
	}
}

func GetAPICreatorPages(website, creatorId string) string {
	if website == Fantia {
		return "https://fantia.jp/fanclubs/" + creatorId + "/posts"
	} else if website == PixivFanbox {
		return "https://api.fanbox.cc/post.paginateCreator"
	} else {
		panic("invalid website")
	}
}

func GetPostDetails(postIds []string, website string, cookies []http.Cookie) ([]map[string]string, []map[string]string) {
	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if len(postIds) < maxConcurrency {
		maxConcurrency = len(postIds)
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, len(postIds))
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
			<-queue
		}(postId)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	// parse the responses
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
	}
	return urlsMap, gdriveUrls
}

func GetCreatorsPosts(creatorIds []string, website string, cookies []http.Cookie) []string {
	var postIds []string
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
		}
	} else {
		panic("invalid website")
	}
	return postIds
}