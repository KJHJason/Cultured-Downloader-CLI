package pixivweb

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/ugoira"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)


// Process the artwork details JSON and returns a map of urls
// with its file path or a Ugoira struct (One of them will be null depending on the artworkType)
func processArtworkJson(res *http.Response, artworkType int64, postDownloadDir string) ([]*request.ToDownload, *models.Ugoira, error) {
	if artworkType == UGOIRA {
		var ugoiraJson models.PixivWebArtworkUgoiraJson
		err := utils.LoadJsonFromResponse(res, &ugoiraJson)
		if err != nil {
			return nil, nil, err
		}

		ugoiraMap := ugoiraJson.Body
		originalUrl := ugoiraMap.OriginalSrc
		ugoiraInfo := &models.Ugoira{
			Url:      originalUrl,
			FilePath: postDownloadDir,
			Frames:   ugoira.MapDelaysToFilename(ugoiraMap.Frames),
		}
		return nil, ugoiraInfo, nil
	}

	var artworkUrls models.PixivWebArtworkJson
	err := utils.LoadJsonFromResponse(res, &artworkUrls)
	if err != nil {
		return nil, nil, err
	}

	var urlsToDownload []*request.ToDownload
	for _, artworkUrl := range artworkUrls.Body {
		urlsToDownload = append(urlsToDownload, &request.ToDownload{
			Url:      artworkUrl.Urls.Original,
			FilePath: postDownloadDir,
		})
	}
	return urlsToDownload, nil, nil
}

// Process the tag search results JSON and returns a slice of artwork IDs
func processTagJsonResults(res *http.Response) ([]string, error) {
	var pixivTagJson models.PixivTag
	resBody, err := utils.ReadResBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resBody, &pixivTagJson)
	if err != nil {
		err = fmt.Errorf(
			"pixiv error %d: failed to unmarshal json for Pixiv's Tag JSON due to %v\nJSON: %s",
			utils.JSON_ERROR,
			err,
			string(resBody),
		)
		return nil, err
	}

	artworksSlice := []string{}
	for _, illust := range pixivTagJson.Body.IllustManga.Data {
		artworksSlice = append(artworksSlice, illust.Id)
	}
	return artworksSlice, nil
}
