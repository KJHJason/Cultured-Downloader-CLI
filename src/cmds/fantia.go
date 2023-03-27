package cmds

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/cmds/textparser"
	"github.com/spf13/cobra"
)

var (
	fantiaDlTextFile    string
	fantiaCookieFile    string
	fantiaSession       string
	fantiaFanclubIds    []string
	fantiaPageNums      []string
	fantiaPostIds       []string
	fantiaDlGdrive      bool
	fantiaGdriveApiKey  string
	fantiaDlThumbnails  bool
	fantiaDlImages      bool
	fantiaDlAttachments bool
	fantiaOverwrite     bool
	fantiaLogUrls       bool
	fantiaUserAgent     string
	fantiaCmd           = &cobra.Command{
		Use:   "fantia",
		Short: "Download from Fantia",
		Long:  "Supports downloads from Fantia Fanclubs and individual posts.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			if fantiaDlTextFile != "" {
				postIds, fanclubInfoSlice := textparser.ParseFantiaTextFile(fantiaDlTextFile)
				fantiaPostIds = append(fantiaPostIds, postIds...)

				for _, fanclubInfo := range fanclubInfoSlice {
					fantiaFanclubIds = append(fantiaFanclubIds, fanclubInfo.FanclubId)
					fantiaPageNums = append(fantiaPageNums, fanclubInfo.PageNum)
				}
			}

			fantiaConfig := &configs.Config{
				OverwriteFiles: fantiaOverwrite,
				UserAgent:      fantiaUserAgent,
				LogUrls:        fantiaLogUrls,
			}

			var gdriveClient *gdrive.GDrive
			if fantiaGdriveApiKey != "" {
				gdriveClient = gdrive.GetNewGDrive(
					fantiaGdriveApiKey,
					fantiaConfig,
					utils.MAX_CONCURRENT_DOWNLOADS,
				)
			}

			fantiaDl := &fantia.FantiaDl{
				FanclubIds:      fantiaFanclubIds,
				FanclubPageNums: fantiaPageNums,
				PostIds:         fantiaPostIds,
			}
			fantiaDl.ValidateArgs()

			fantiaDlOptions := &fantia.FantiaDlOptions{
				DlThumbnails:    fantiaDlThumbnails,
				DlImages:        fantiaDlImages,
				DlAttachments:   fantiaDlAttachments,
				DlGdrive:        fantiaDlGdrive,
				GdriveClient:    gdriveClient,
				Configs:         fantiaConfig,
				SessionCookieId: fantiaSession,
			}
			if fantiaCookieFile != "" {
				cookies, err := utils.ParseNetscapeCookieFile(
					fantiaCookieFile,
					fantiaSession,
					utils.FANTIA,
				)
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
						utils.ERROR,
					)
				}
				fantiaDlOptions.SessionCookies = cookies
			}

			err := fantiaDlOptions.ValidateArgs(fantiaUserAgent)
			if err != nil {
				utils.LogError(
					err,
					"",
					true,
					utils.ERROR,
				)
			}

			utils.PrintWarningMsg()
			fantia.FantiaDownloadProcess(
				fantiaDl,
				fantiaDlOptions,
			)
		},
	}
)

func init() {
	mutlipleIdsMsg := getMultipleIdsMsg()
	fantiaCmd.Flags().StringVarP(
		&fantiaSession,
		"session",
		"s",
		"",
		"Your \"_session_id\" cookie value to use for the requests to Fantia.",
	)
	fantiaCmd.Flags().StringSliceVar(
		&fantiaFanclubIds,
		"fanclub_id",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Fantia Fanclub ID(s) to download from.",
				mutlipleIdsMsg,
			},
		),
	)
	fantiaCmd.Flags().StringSliceVar(
		&fantiaPageNums,
		"page_num",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied Fantia Fanclub ID(s).",
				"Format: \"num\", \"minNum-maxNum\", or \"\" to download all pages",
				"Leave blank to download all pages from each Fantia Fanclub.",
			},
		),
	)
	fantiaCmd.Flags().StringSliceVar(
		&fantiaPostIds,
		"post_id",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Fantia post ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)
	fantiaCmd.Flags().BoolVar(
		&fantiaDlGdrive,
		"dl_gdrive",
		true,
		"Whether to download the Google Drive links of a post on Fantia.",
	)
	fantiaCmd.Flags().BoolVar(
		&fantiaDlThumbnails,
		"dl_thumbnails",
		true,
		"Whether to download the thumbnail of a post on Fantia.",
	)
	fantiaCmd.Flags().BoolVar(
		&fantiaDlImages,
		"dl_images",
		true,
		"Whether to download the images of a post on Fantia.",
	)
	fantiaCmd.Flags().BoolVar(
		&fantiaDlAttachments,
		"dl_attachments",
		true,
		"Whether to download the attachments of a post on Fantia.",
	)
}
