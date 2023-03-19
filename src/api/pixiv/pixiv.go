package pixiv

import (
	"fmt"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/ugoira"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/common"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/web"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/mobile"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Start the download process for Pixiv
func PixivWebDownloadProcess(config *configs.Config, pixivDl *PixivDl, pixivDlOptions *pixivweb.PixivWebDlOptions, pixivUgoiraOptions *ugoira.UgoiraOptions) {
	var ugoiraToDl []*models.Ugoira
	var artworksToDl []*request.ToDownload
	if len(pixivDl.IllustratorIds) > 0 {
		artworkIdsSlice := pixivweb.GetMultipleIllustratorPosts(
			pixivDl.IllustratorIds,
			pixivDl.IllustratorPageNums,
			utils.DOWNLOAD_PATH,
			config,
			pixivDlOptions,
		)
		pixivDl.ArtworkIds = append(pixivDl.ArtworkIds, artworkIdsSlice...)
		pixivDl.ArtworkIds = utils.RemoveSliceDuplicates(pixivDl.ArtworkIds)
	}

	if len(pixivDl.ArtworkIds) > 0 {
		artworkSlice, ugoiraSlice := pixivweb.GetMultipleArtworkDetails(
			pixivDl.ArtworkIds,
			utils.DOWNLOAD_PATH,
			config,
			pixivDlOptions.SessionCookies,
		)
		artworksToDl = append(artworksToDl, artworkSlice...)
		ugoiraToDl = append(ugoiraToDl, ugoiraSlice...)
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
			artworksSlice, ugoiraSlice, hasErr = pixivweb.TagSearch(
				tagName,
				utils.DOWNLOAD_PATH,
				pixivDl.TagNamesPageNums[idx],
				config,
				pixivDlOptions,
			)
			artworksToDl = append(artworksToDl, artworksSlice...)
			ugoiraToDl = append(ugoiraToDl, ugoiraSlice...)
			progress.MsgIncrement(baseMsg)
		}
		progress.Stop(hasErr)
	}

	if len(artworksToDl) > 0 {
		request.DownloadUrls(
			artworksToDl,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Headers:        pixivcommon.GetPixivRequestHeaders(),
				Cookies:        pixivDlOptions.SessionCookies,
				UseHttp3:       false,
			},
			config,
		)
	}
	if len(ugoiraToDl) > 0 {
		ugoira.DownloadMultipleUgoira(
			&ugoira.UgoiraArgs{
				UseMobileApi: false,
				ToDownload:   ugoiraToDl,
				Cookies:      pixivDlOptions.SessionCookies,
			},
			pixivUgoiraOptions,
			config,
			request.CallRequest,
		)
	}
}

// Start the download process for Pixiv
func PixivMobileDownloadProcess(config *configs.Config, pixivDl *PixivDl, pixivDlOptions *pixivmobile.PixivMobileDlOptions, pixivUgoiraOptions *ugoira.UgoiraOptions) {
	var ugoiraToDl []*models.Ugoira
	var artworksToDl []*request.ToDownload
	if len(pixivDl.IllustratorIds) > 0 {
		artworkSlice, ugoiraSlice := pixivDlOptions.MobileClient.GetMultipleIllustratorPosts(
			pixivDl.IllustratorIds,
			pixivDl.IllustratorPageNums,
			utils.DOWNLOAD_PATH,
			pixivDlOptions.ArtworkType,
		)
		artworksToDl = artworkSlice
		ugoiraToDl = ugoiraSlice
	}

	if len(pixivDl.ArtworkIds) > 0 {
		artworkSlice, ugoiraSlice := pixivDlOptions.MobileClient.GetMultipleArtworkDetails(
			pixivDl.ArtworkIds,
			utils.DOWNLOAD_PATH,
		)
		artworksToDl = append(artworksToDl, artworkSlice...)
		ugoiraToDl = append(ugoiraToDl, ugoiraSlice...)
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
			artworksSlice, ugoiraSlice, hasErr = pixivDlOptions.MobileClient.TagSearch(
				tagName,
				utils.DOWNLOAD_PATH,
				pixivDl.TagNamesPageNums[idx],
				pixivDlOptions,
			)
			artworksToDl = append(artworksToDl, artworksSlice...)
			ugoiraToDl = append(ugoiraToDl, ugoiraSlice...)
			progress.MsgIncrement(baseMsg)
		}
		progress.Stop(hasErr)
	}

	if len(artworksToDl) > 0 {
		request.DownloadUrls(
			artworksToDl,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Headers:        pixivcommon.GetPixivRequestHeaders(),
				UseHttp3:       false,
			},
			config,
		)
	}
	if len(ugoiraToDl) > 0 {
		ugoira.DownloadMultipleUgoira(
			&ugoira.UgoiraArgs{
				UseMobileApi: true,
				ToDownload:   ugoiraToDl,
				Cookies:      nil,
			},
			pixivUgoiraOptions,
			config,
			pixivDlOptions.MobileClient.SendRequest,
		)
	}
}
