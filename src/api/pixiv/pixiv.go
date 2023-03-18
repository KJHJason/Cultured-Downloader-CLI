package pixiv

import (
	"fmt"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Start the download process for Pixiv
func PixivDownloadProcess(config *configs.Config, pixivDl *PixivDl, pixivDlOptions *PixivDlOptions, pixivUgoiraOptions *UgoiraOptions) {
	var ugoiraToDl []*models.Ugoira
	var artworksToDl []*request.ToDownload
	if len(pixivDl.IllustratorIds) > 0 {
		if pixivDlOptions.MobileClient == nil {
			artworkIdsSlice := getMultipleIllustratorPosts(
				pixivDl.IllustratorIds,
				pixivDl.IllustratorPageNums,
				utils.DOWNLOAD_PATH,
				config,
				pixivDlOptions,
			)
			pixivDl.ArtworkIds = append(pixivDl.ArtworkIds, artworkIdsSlice...)
			pixivDl.ArtworkIds = utils.RemoveSliceDuplicates(pixivDl.ArtworkIds)
		} else {
			artworkSlice, ugoiraSlice := pixivDlOptions.MobileClient.getMultipleIllustratorPosts(
				pixivDl.IllustratorIds,
				pixivDl.IllustratorPageNums,
				utils.DOWNLOAD_PATH,
				pixivDlOptions.ArtworkType,
			)
			artworksToDl = artworkSlice
			ugoiraToDl = ugoiraSlice
		}
	}

	if len(pixivDl.ArtworkIds) > 0 {
		var artworkSlice []*request.ToDownload
		var ugoiraSlice []*models.Ugoira
		if pixivDlOptions.MobileClient == nil {
			artworkSlice, ugoiraSlice = getMultipleArtworkDetails(
				pixivDl.ArtworkIds,
				utils.DOWNLOAD_PATH,
				config,
				pixivDlOptions.SessionCookies,
			)
		} else {
			artworkSlice, ugoiraSlice = pixivDlOptions.MobileClient.getMultipleArtworkDetails(
				pixivDl.ArtworkIds,
				utils.DOWNLOAD_PATH,
			)
		}

		if len(artworksToDl) > 0 {
			artworksToDl = append(artworksToDl, artworkSlice...)
		} else {
			artworksToDl = artworkSlice
		}
		if len(ugoiraToDl) > 0 {
			ugoiraToDl = append(ugoiraToDl, ugoiraSlice...)
		} else {
			ugoiraToDl = ugoiraSlice
		}
	}

	if len(pixivDl.TagNames) > 0 {
		// loop through each tag and page number
		baseMsg := "Searching for artworks based on tag names on Pixiv [%d/" + fmt.Sprintf("%d]...", len(pixivDl.TagNames))
		progress := spinner.New(
			"pong",
			"fgHiYellow",
			fmt.Sprintf(
				baseMsg,
				0,
			),
			fmt.Sprintf(
				"Finished searching for artworks based on %d tag names on Pixiv!",
				len(pixivDl.TagNames),
			),
			fmt.Sprintf(
				"Finished with some errors while searching for artworks based on %d tag names on Pixiv!\nPlease refer to the logs for more details...",
				len(pixivDl.TagNames),
			),
			len(pixivDl.TagNames),
		)
		progress.Start()
		hasErr := false
		for idx, tagName := range pixivDl.TagNames {
			var artworksSlice []*request.ToDownload
			var ugoiraSlice []*models.Ugoira
			if pixivDlOptions.MobileClient == nil {
				artworksSlice, ugoiraSlice, hasErr = tagSearch(
					tagName,
					utils.DOWNLOAD_PATH,
					pixivDl.TagNamesPageNums[idx],
					config,
					pixivDlOptions,
				)
			} else {
				artworksSlice, ugoiraSlice, hasErr = pixivDlOptions.MobileClient.tagSearch(
					tagName,
					utils.DOWNLOAD_PATH,
					pixivDl.TagNamesPageNums[idx],
					pixivDlOptions,
				)
			}
			artworksToDl = append(artworksToDl, artworksSlice...)
			ugoiraToDl = append(ugoiraToDl, ugoiraSlice...)
			progress.MsgIncrement(baseMsg)
		}
		progress.Stop(hasErr)
	}

	if len(artworksToDl) > 0 {
		headers := GetPixivRequestHeaders()
		request.DownloadUrls(
			artworksToDl,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Headers:        headers,
				Cookies:        pixivDlOptions.SessionCookies,
				UseHttp3:       false,
			},
			config,
		)
	}
	if len(ugoiraToDl) > 0 {
		downloadMultipleUgoira(
			ugoiraToDl,
			config,
			pixivUgoiraOptions,
			pixivDlOptions.SessionCookies,
		)
	}
}
