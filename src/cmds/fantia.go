package cmds

import (
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	fantiaCookieFile    string
	fantiaSession       string
	fantiaFanclubIds    []string
	fantiaPageNums      []string
	fantiaPostIds       []string
	fantiaDlThumbnails  bool
	fantiaDlImages      bool
	fantiaDlAttachments bool
	fantiaOverwrite     bool
	fantiaCmd           = &cobra.Command{
		Use:   "fantia",
		Short: "Download from Fantia",
		Long:  "Supports downloading from Fantia Fanclubs and individual posts.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			fantiaConfig := api.Config{
				OverwriteFiles: fantiaOverwrite,
			}
			fantiaDl := fantia.FantiaDl{
				FanclubIds:      fantiaFanclubIds,
				FanclubPageNums: fantiaPageNums,
				PostIds:         fantiaPostIds,
			}
			fantiaDl.ValidateArgs()

			fantiaDlOptions := fantia.FantiaDlOptions{
				DlThumbnails:    fantiaDlThumbnails,
				DlImages:        fantiaDlImages,
				DlAttachments:   fantiaDlAttachments,
				SessionCookieId: fantiaSession,
			}
			if fantiaCookieFile != "" {
				cookies, err := utils.ParseNetscapeCookieFile(fantiaCookieFile, utils.FANTIA, fantiaSession)
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
					)
				}
				fantiaDlOptions.SessionCookies = cookies
			}

			err := fantiaDlOptions.ValidateArgs()
			if err != nil {
				utils.LogError(
					err,
					"",
					true,
				)
			}

			fantia.FantiaDownloadProcess(
				&fantiaConfig,
				&fantiaDl,
				&fantiaDlOptions,
			)
		},
	}
)
