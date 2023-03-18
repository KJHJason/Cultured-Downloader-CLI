package fantia

import "github.com/KJHJason/Cultured-Downloader-CLI/configs"

// Start the download process for Fantia
func FantiaDownloadProcess(config *configs.Config, fantiaDl *FantiaDl, fantiaDlOptions *FantiaDlOptions) {
	if !fantiaDlOptions.DlThumbnails && !fantiaDlOptions.DlImages && !fantiaDlOptions.DlAttachments {
		return
	}

	if len(fantiaDl.FanclubIds) > 0 {
		fantiaDl.getCreatorsPosts(
			config,
			fantiaDlOptions,
		)
	}
	if len(fantiaDl.PostIds) > 0 {
		fantiaDl.dlFantiaPosts(
			fantiaDlOptions,
			config,
		)
	}
}
