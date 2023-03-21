package fantia

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

// Start the download process for Fantia
func FantiaDownloadProcess(config *configs.Config, fantiaDl *FantiaDl, fantiaDlOptions *FantiaDlOptions) {
	if !fantiaDlOptions.DlThumbnails && !fantiaDlOptions.DlImages && !fantiaDlOptions.DlAttachments {
		return
	}

	if len(fantiaDl.FanclubIds) > 0 {
		fantiaDl.getCreatorsPosts(fantiaDlOptions)
	}
	var gdriveLinks []*request.ToDownload
	if len(fantiaDl.PostIds) > 0 {
		fantiaDl.dlFantiaPosts(fantiaDlOptions)
	}

	if fantiaDlOptions.GdriveClient != nil && len(gdriveLinks) > 0 {
		fantiaDlOptions.GdriveClient.DownloadGdriveUrls(gdriveLinks, config)
	}
}
