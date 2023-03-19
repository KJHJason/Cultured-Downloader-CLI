package kemono

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func KemonoDownloadProcess(config *configs.Config, kemonoDl *KemonoDl, kemonoDlOptions *KemonoDlOptions, dlFav bool) {
	if !kemonoDlOptions.DlAttachments && !kemonoDlOptions.DlGdrive {
		return
	}

	var toDownload, gdriveLinks []*request.ToDownload
	if dlFav {
		progress := spinner.New(
			spinner.REQ_SPINNER,
			"fgHiYellow",
			"Getting favourites from Kemono Party...",
			"Finished getting favourites from Kemono Party!",
			"Something went wrong while getting favourites from Kemono Party.\nPlease refer to the logs for more details.",
			0,
		)
		progress.Start()
		favToDl, favGdriveLinks, err := getFavourites(
			config,
			utils.DOWNLOAD_PATH,
			kemonoDlOptions,
		)
		hasErr := (err != nil)
		if hasErr {
			utils.LogError(err, "", false, utils.ERROR)
		} else {
			toDownload = favToDl
			gdriveLinks = favGdriveLinks
		}
		progress.Stop(hasErr)
	}

	if len(kemonoDl.PostsToDl) > 0 {
		postsToDl, gdriveLinksToDl := getMultiplePosts(
			config,
			kemonoDl.PostsToDl,
			utils.DOWNLOAD_PATH,
			kemonoDlOptions,
		)
		toDownload = append(toDownload, postsToDl...)
		gdriveLinks = append(gdriveLinks, gdriveLinksToDl...)
	}
	if len(kemonoDl.CreatorsToDl) > 0 {
		creatorsToDl, gdriveLinksToDl := getMultipleCreators(
			config,
			kemonoDl.CreatorsToDl,
			utils.DOWNLOAD_PATH,
			kemonoDlOptions,
		)
		toDownload = append(toDownload, creatorsToDl...)
		gdriveLinks = append(gdriveLinks, gdriveLinksToDl...)
	}

	if len(toDownload) > 0 {
		request.DownloadUrls(
			toDownload,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Cookies:        kemonoDlOptions.SessionCookies,
				UseHttp3:       utils.IsHttp3Supported(utils.KEMONO, false),
			},
			config,
		)
	}
	if kemonoDlOptions.GdriveClient != nil && len(gdriveLinks) > 0 {
		kemonoDlOptions.GdriveClient.DownloadGdriveUrls(gdriveLinks, config)
	}
}
