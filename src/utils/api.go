package utils

import (
	"fmt"
	"sync"
	"net/http"
	"github.com/panjf2000/ants/v2"
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

func GetPostDetails(postIdsOrUrls []string, website string, cookies []http.Cookie) ([]map[string]string, []map[string]string) {
	var wg sync.WaitGroup
	maxConcurrency := MAX_CONCURRENT_DOWNLOADS
	if len(postIdsOrUrls) < MAX_CONCURRENT_DOWNLOADS {
		maxConcurrency = len(postIdsOrUrls)
	}
	pool, _ := ants.NewPool(maxConcurrency)
	resChan := make(chan *http.Response, len(postIdsOrUrls))
	for _,  postIdOrUrl := range postIdsOrUrls {
		wg.Add(1)
		err := pool.Submit(func() {
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

			res, err := CallRequest(url, 30, cookies, "GET", header, params, false)
			if err != nil || res.StatusCode != 200 {
				LogError(err, fmt.Sprintf("failed to get post details for %s", url), false)
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

	// parse the responses
	var urlsMap []map[string]string
	var gdriveUrls []map[string]string
	for res := range resChan {
		if website == "fantia" {
			urlsMap = append(urlsMap, ProcessFantiaPost(res, DOWNLOAD_PATH)...)
		} else if website == "fanbox" {
			postUrls, postGdriveLinks := ProcessFanboxPost(res, nil, DOWNLOAD_PATH)
			urlsMap = append(urlsMap, postUrls...)
			gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		} else {
			panic("invalid website")
		}
	}
	return urlsMap, gdriveUrls
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
		pool, _ := ants.NewPool(maxConcurrency)
		resChan := make(chan []string, len(creatorIds))
		for _, creatorId := range creatorIds {
			wg.Add(1)
			err := pool.Submit(func() {
				defer wg.Done()
				resChan <- GetFantiaPosts(creatorId, cookies)
			})
			if err != nil {
				panic(err)
			}
		}
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