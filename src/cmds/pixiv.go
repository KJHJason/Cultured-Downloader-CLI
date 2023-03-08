package cmds

import (
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	pixivCookieFile     string
	pixivFfmpegPath     string
	pixivStartOauth     bool
	pixivRefreshToken   string
	pixivSession        string
	deleteUgoiraZip     bool
	ugoiraQuality       int
	ugoiraOutputFormat  string
	pixivArtworkIds     []string
	pixivIllustratorIds []string
	pixivTagNames       []string
	pixivPageNums       []string
	pixivSortOrder      string
	pixivSearchMode     string
	pixivRatingMode     string
	pixivArtworkType    string
	pixivOverwrite      bool
	pixivCmd            = &cobra.Command{
		Use:   "pixiv",
		Short: "Download from Pixiv",
		Long:  "Supports downloading from Pixiv by artwork ID, illustrator ID, tag name, and more.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			if pixivStartOauth {
				err := pixiv.NewPixivMobile("", 10).StartOauthFlow()
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
					)
				}
				return
			}

			pixivConfig := api.Config{
				FfmpegPath:     pixivFfmpegPath,
				OverwriteFiles: pixivOverwrite,
			}
			pixivConfig.ValidateFfmpeg()

			utils.ValidatePageNumInput(
				len(pixivTagNames),
				&pixivPageNums,
				[]string{
					"Number of tag names and page numbers must be equal.",
				},
			)
			pixivDl := pixiv.PixivDl{
				ArtworkIds:       pixivArtworkIds,
				IllustratorIds:   pixivIllustratorIds,
				TagNames:         pixivTagNames,
				TagNamesPageNums: pixivPageNums,
			}
			pixivDl.ValidateArgs()

			pixivUgoiraOptions := pixiv.UgoiraOptions{
				DeleteZip:    deleteUgoiraZip,
				Quality:      ugoiraQuality,
				OutputFormat: ugoiraOutputFormat,
			}
			pixivUgoiraOptions.ValidateArgs()

			pixivDlOptions := pixiv.PixivDlOptions{
				SortOrder:       pixivSortOrder,
				SearchMode:      pixivSearchMode,
				RatingMode:      pixivRatingMode,
				ArtworkType:     pixivArtworkType,
				RefreshToken:    pixivRefreshToken,
				SessionCookieId: pixivSession,
			}
			if pixivCookieFile != "" {
				cookies, err := utils.ParseNetscapeCookieFile(pixivCookieFile, utils.PIXIV, pixivSession)
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
					)
				}
				pixivDlOptions.SessionCookies = cookies
			}
			pixivDlOptions.ValidateArgs()

			pixiv.PixivDownloadProcess(
				&pixivConfig,
				&pixivDl,
				&pixivDlOptions,
				&pixivUgoiraOptions,
			)
		},
	}
)
