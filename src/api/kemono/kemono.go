package kemono

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func KemonoDownloadProcess(config *configs.Config, kemonoDl *KemonoDl, kemonoDlOptions *KemonoDlOptions, dlFav bool) {
	if !kemonoDlOptions.DlAttachments && !kemonoDlOptions.DlGdrive {
		return
	}

	var toDownload, gdriveLinks []*request.ToDownload
	if dlFav {
		favToDl, favGdriveLinks, err := getFavourites(
			config,
			utils.DOWNLOAD_PATH,
			kemonoDlOptions,
		)
		if err != nil {
			utils.LogError(err, "", false, utils.ERROR)
		} else {
			toDownload = favToDl
			gdriveLinks = favGdriveLinks
		}
	}

	if len(kemonoDl.PostsToDl) > 0 {
		getMultiplePosts(
			config,
			kemonoDl.PostsToDl,
			utils.DOWNLOAD_PATH,
			kemonoDlOptions,
		)
	}
	if len(kemonoDl.CreatorsToDl) > 0 {
		getMultipleCreators(
			config,
			kemonoDl.CreatorsToDl,
			utils.DOWNLOAD_PATH,
			kemonoDlOptions,
		)
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
