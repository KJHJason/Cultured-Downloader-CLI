package kemono

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func KemonoDownloadProcess(config *configs.Config, kemonoDl *KemonoDl, dlOptions *KemonoDlOptions, dlFav bool) {
	if !dlOptions.DlAttachments && !dlOptions.DlGdrive {
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
			utils.DOWNLOAD_PATH,
			dlOptions,
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
			kemonoDl.PostsToDl,
			utils.DOWNLOAD_PATH,
			dlOptions,
		)
		toDownload = append(toDownload, postsToDl...)
		gdriveLinks = append(gdriveLinks, gdriveLinksToDl...)
	}
	if len(kemonoDl.CreatorsToDl) > 0 {
		creatorsToDl, gdriveLinksToDl := getMultipleCreators(
			kemonoDl.CreatorsToDl,
			utils.DOWNLOAD_PATH,
			dlOptions,
		)
		toDownload = append(toDownload, creatorsToDl...)
		gdriveLinks = append(gdriveLinks, gdriveLinksToDl...)
	}

	var downloadedPosts bool
	if len(toDownload) > 0 {
		downloadedPosts = true
		request.DownloadUrls(
			toDownload,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Cookies:        dlOptions.SessionCookies,
				UseHttp3:       utils.IsHttp3Supported(utils.KEMONO, false),
			},
			config,
		)
	}
	if dlOptions.GdriveClient != nil && len(gdriveLinks) > 0 {
		downloadedPosts = true
		dlOptions.GdriveClient.DownloadGdriveUrls(gdriveLinks, config)
	}

	if downloadedPosts {
		utils.AlertWithoutErr(utils.Title, "Downloaded all posts from Kemono Party!")
	} else {
		utils.AlertWithoutErr(utils.Title, "No posts to download from Kemono Party!")
	}
}
