package cmds

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/cmds/textparser"
	"github.com/spf13/cobra"
)

var (
	fanboxDlTextFile     string
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
	fanboxLogUrls        bool
	fanboxUserAgent      string
	pixivFanboxCmd       = &cobra.Command{
		Use:   "pixiv_fanbox",
		Short: "Download from Pixiv Fanbox",
		Long:  "Supports downloads from Pixiv Fanbox creators and individual posts.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			pixivFanboxConfig := &configs.Config{
				OverwriteFiles: fanboxOverwriteFiles,
				UserAgent:      fanboxUserAgent,
				LogUrls:        fanboxLogUrls,
			}
			var gdriveClient *gdrive.GDrive
			if fanboxGdriveApiKey != "" {
				gdriveClient = gdrive.GetNewGDrive(
					fanboxGdriveApiKey,
					pixivFanboxConfig,
					utils.MAX_CONCURRENT_DOWNLOADS,
				)
			}

			if fanboxDlTextFile != "" {
				postIds, creatorInfoSlice := textparser.ParsePixivFanboxTextFile(fanboxDlTextFile)
				fanboxPostIds = append(fanboxPostIds, postIds...)

				for _, creatorInfo := range creatorInfoSlice {
					fanboxCreatorIds = append(fanboxCreatorIds, creatorInfo.CreatorId)
					fanboxPageNums = append(fanboxPageNums, creatorInfo.PageNum)
				}
			}
			pixivFanboxDl := &pixivfanbox.PixivFanboxDl{
				CreatorIds:      fanboxCreatorIds,
				CreatorPageNums: fanboxPageNums,
				PostIds:         fanboxPostIds,
			}
			pixivFanboxDl.ValidateArgs()

			pixivFanboxDlOptions := &pixivfanbox.PixivFanboxDlOptions{
				DlThumbnails:    fanboxDlThumbnails,
				DlImages:        fanboxDlImages,
				DlAttachments:   fanboxDlAttachments,
				Configs:         pixivFanboxConfig,
				GdriveClient:    gdriveClient,
				DlGdrive:        fanboxDlGdrive,
				SessionCookieId: fanboxSession,
			}
			if fanboxCookieFile != "" {
				cookies, err := utils.ParseNetscapeCookieFile(
					fanboxCookieFile,
					fanboxSession,
					utils.PIXIV_FANBOX,
				)
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
						utils.ERROR,
					)
				}
				pixivFanboxDlOptions.SessionCookies = cookies
			}
			pixivFanboxDlOptions.ValidateArgs(fanboxUserAgent)

			utils.PrintWarningMsg()
			pixivfanbox.PixivFanboxDownloadProcess(
				pixivFanboxDl,
				pixivFanboxDlOptions,
			)
		},
	}
)

func init() {
	mutlipleIdsMsg := getMultipleIdsMsg()
	pixivFanboxCmd.Flags().StringVarP(
		&fanboxSession,
		"session",
		"s",
		"",
		"Your \"FANBOXSESSID\" cookie value to use for the requests to Pixiv Fanbox.",
	)
	pixivFanboxCmd.Flags().StringSliceVar(
		&fanboxCreatorIds,
		"creator_id",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Pixiv Fanbox Creator ID(s) to download from.",
				mutlipleIdsMsg,
			},
		),
	)
	pixivFanboxCmd.Flags().StringSliceVar(
		&fanboxPageNums,
		"page_num",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied Pixiv Fanbox creator ID(s).",
				"Format: \"num\", \"minNum-maxNum\", or \"\" to download all pages",
				"Leave blank to download all pages from each creator.",
			},
		),
	)
	pixivFanboxCmd.Flags().StringSliceVar(
		&fanboxPostIds,
		"post_id",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Pixiv Fanbox post ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)
	pixivFanboxCmd.Flags().BoolVarP(
		&fanboxDlThumbnails,
		"dl_thumbnails",
		"t",
		true,
		"Whether to download the thumbnail of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().BoolVarP(
		&fanboxDlImages,
		"dl_images",
		"i",
		true,
		"Whether to download the images of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().BoolVarP(
		&fanboxDlAttachments,
		"dl_attachments",
		"a",
		true,
		"Whether to download the attachments of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().BoolVarP(
		&fanboxDlGdrive,
		"dl_gdrive",
		"g",
		true,
		"Whether to download the Google Drive links of a Pixiv Fanbox post.",
	)
}
