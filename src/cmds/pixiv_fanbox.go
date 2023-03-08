package cmds

import (
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	fanboxCookieFile     string
	fanboxSession        string
	fanboxCreatorIds     []string
	fanboxPageNums       []string
	fanboxPostIds        []string
	fanboxDlThumbnails   bool
	fanboxDlImages       bool
	fanboxDlAttachments  bool
	fanboxDlGdrive       bool
	fanboxGdriveApiKey   string
	fanboxOverwriteFiles bool
	pixivFanboxCmd       = &cobra.Command{
		Use:   "pixiv_fanbox",
		Short: "Download from Pixiv Fanbox",
		Long:  "Supports downloading from Pixiv by artwork ID, illustrator ID, tag name, and more.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			pixivFanboxConfig := api.Config{
				OverwriteFiles: fanboxOverwriteFiles,
			}
			if fanboxGdriveApiKey != "" {
				pixivFanboxConfig.GDriveClient = gdrive.GetNewGDrive(fanboxGdriveApiKey, utils.MAX_CONCURRENT_DOWNLOADS)
			}

			pixivFanboxDl := pixivfanbox.PixivFanboxDl{
				CreatorIds:      fanboxCreatorIds,
				CreatorPageNums: fanboxPageNums,
				PostIds:         fanboxPostIds,
			}
			pixivFanboxDl.ValidateArgs()

			pixivFanboxDlOptions := pixivfanbox.PixivFanboxDlOptions{
				DlThumbnails:    fanboxDlThumbnails,
				DlImages:        fanboxDlImages,
				DlAttachments:   fanboxDlAttachments,
				DlGdrive:        fanboxDlGdrive,
				SessionCookieId: fanboxSession,
			}
			if fanboxCookieFile != "" {
				cookies, err := utils.ParseNetscapeCookieFile(fanboxCookieFile, utils.PIXIV_FANBOX, fanboxSession)
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
					)
				}
				pixivFanboxDlOptions.SessionCookies = cookies
			}
			pixivFanboxDlOptions.ValidateArgs()

			pixivfanbox.PixivFanboxDownloadProcess(
				&pixivFanboxConfig,
				&pixivFanboxDl,
				&pixivFanboxDlOptions,
			)
		},
	}
)
