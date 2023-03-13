package cmds

import (
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	fantiaDlTextFile    string
	fantiaCookieFile    string
	fantiaSession       string
	fantiaFanclubIds    []string
	fantiaPageNums      []string
	fantiaPostIds       []string
	fantiaDlThumbnails  bool
	fantiaDlImages      bool
	fantiaDlAttachments bool
	fantiaOverwrite     bool
	fantiaUserAgent     string
	fantiaCmd           = &cobra.Command{
		Use:   "fantia",
		Short: "Download from Fantia",
		Long:  "Supports downloading from Fantia Fanclubs and individual posts.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			if fantiaDlTextFile != "" {
				postIds, fanclubInfoSlice := parseFantiaTextFile(fantiaDlTextFile)
				fantiaPostIds = append(fantiaPostIds, postIds...)

				for _, fanclubInfo := range fanclubInfoSlice {
					fantiaFanclubIds = append(fantiaFanclubIds, fanclubInfo.FanclubId)
					fantiaPageNums = append(fantiaPageNums, fanclubInfo.PageNum)
				}
			}

			fantiaConfig := api.Config{
				OverwriteFiles: fantiaOverwrite,
				UserAgent:      fantiaUserAgent,
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
				)
			}

			utils.PrintWarningMsg()
			fantia.FantiaDownloadProcess(
				&fantiaConfig,
				&fantiaDl,
				&fantiaDlOptions,
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
		"Your _session_id cookie value to use for the requests to Fantia.",
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
		&fantiaDlThumbnails,
		"dl_thumbnails",
		true,
		"Whether to download the thumbnail of a Fantia post.",
	)
	fantiaCmd.Flags().BoolVar(
		&fantiaDlImages,
		"dl_images",
		true,
		"Whether to download the images of a Fantia post.",
	)
	fantiaCmd.Flags().BoolVar(
		&fantiaDlAttachments,
		"dl_attachments",
		true,
		"Whether to download the attachments of a Fantia post.",
	)
}
