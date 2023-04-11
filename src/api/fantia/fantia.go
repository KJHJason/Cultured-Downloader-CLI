package fantia

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Start the download process for Fantia
func FantiaDownloadProcess(fantiaDl *FantiaDl, fantiaDlOptions *FantiaDlOptions) {
	if !fantiaDlOptions.DlThumbnails && !fantiaDlOptions.DlImages && !fantiaDlOptions.DlAttachments {
		return
	}

	if len(fantiaDl.FanclubIds) > 0 {
		fantiaDl.getCreatorsPosts(fantiaDlOptions)
	}

	var gdriveLinks []*request.ToDownload
	var downloadedPosts bool
	if len(fantiaDl.PostIds) > 0 {
		fantiaDl.dlFantiaPosts(fantiaDlOptions)
		downloadedPosts = true
	}

	if fantiaDlOptions.GdriveClient != nil && len(gdriveLinks) > 0 {
		fantiaDlOptions.GdriveClient.DownloadGdriveUrls(gdriveLinks, fantiaDlOptions.Configs)
		downloadedPosts = true
	}

	if downloadedPosts {
		utils.AlertWithoutErr(utils.Title, "Downloaded all posts from Fantia!")
	} else {
		utils.AlertWithoutErr(utils.Title, "No posts to download from Fantia!")
	}
}
