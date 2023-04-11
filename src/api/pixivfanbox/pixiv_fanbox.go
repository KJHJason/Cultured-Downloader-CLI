package pixivfanbox

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Start the download process for Pixiv Fanbox
func PixivFanboxDownloadProcess(pixivFanboxDl *PixivFanboxDl, pixivFanboxDlOptions *PixivFanboxDlOptions) {
	if !pixivFanboxDlOptions.DlThumbnails && !pixivFanboxDlOptions.DlImages && !pixivFanboxDlOptions.DlAttachments && !pixivFanboxDlOptions.DlGdrive {
		return
	}

	if len(pixivFanboxDl.CreatorIds) > 0 {
		pixivFanboxDl.getCreatorsPosts(
			pixivFanboxDlOptions,
		)
	}

	var urlsToDownload, gdriveUrlsToDownload []*request.ToDownload
	if len(pixivFanboxDl.PostIds) > 0 {
		urlsToDownload, gdriveUrlsToDownload = pixivFanboxDl.getPostDetails(
			pixivFanboxDlOptions,
		)
	}

	var downloadedPosts bool
	if len(urlsToDownload) > 0 {
		downloadedPosts = true
		request.DownloadUrls(
			urlsToDownload,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Headers:        GetPixivFanboxHeaders(),
				Cookies:        pixivFanboxDlOptions.SessionCookies,
				UseHttp3:       false,
			},
			pixivFanboxDlOptions.Configs,
		)
	}
	if pixivFanboxDlOptions.GdriveClient != nil && len(gdriveUrlsToDownload) > 0 {
		downloadedPosts = true
		pixivFanboxDlOptions.GdriveClient.DownloadGdriveUrls(gdriveUrlsToDownload, pixivFanboxDlOptions.Configs)
	}

	if downloadedPosts {
		utils.AlertWithoutErr(utils.Title, "Downloaded all posts from Pixiv Fanbox!")
	} else {
		utils.AlertWithoutErr(utils.Title, "No posts to download from Pixiv Fanbox!")
	}
}
