package cmds

import (
	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/cmds/textparser"
	"github.com/spf13/cobra"
)

var (
	kemonoDlTextFile           string
	kemonoCookieFile           string
	kemonoSession              string
	kemonoCreatorUrls          []string
	kemonoPageNums             []string
	kemonoPostUrls             []string
	kemonoDlGdrive             bool
	kemonoGdriveApiKey         string
	kemonoGdriveServiceAccPath string
	kemonoDlAttachments        bool
	kemonoOverwrite            bool
	kemonoLogUrls              bool
	kemonoDlFav                bool
	kemonoUserAgent            string
	kemonoCmd = &cobra.Command{
		Use:   "kemono",
		Short: "Download from Kemono Party",
		Long:  "Supports downloads from creators and posts on Kemono Party.",
		Run: func(cmd *cobra.Command, args []string) {
			kemonoConfig := &configs.Config{
				OverwriteFiles: kemonoOverwrite,
				UserAgent:      kemonoUserAgent,
				LogUrls:        kemonoLogUrls,
			}
			var gdriveClient *gdrive.GDrive
			if kemonoGdriveApiKey != "" {
				gdriveClient = gdrive.GetNewGDrive(
					kemonoGdriveApiKey,
					kemonoGdriveServiceAccPath,
					kemonoConfig,
					utils.MAX_CONCURRENT_DOWNLOADS,
				)
			}

			kemonoDl := &kemono.KemonoDl{
				CreatorUrls:     kemonoCreatorUrls,
				CreatorPageNums: kemonoPageNums,
				PostUrls:        kemonoPostUrls,
			}
			if kemonoDlTextFile != "" {
				kemonoPostToDl, kemonoCreatorToDl := textparser.ParseKemonoTextFile(kemonoDlTextFile)
				kemonoDl.PostsToDl = kemonoPostToDl
				kemonoDl.CreatorsToDl = kemonoCreatorToDl
			}
			kemonoDl.ValidateArgs()

			kemonoDlOptions := &kemono.KemonoDlOptions{
				DlAttachments:   kemonoDlAttachments,
				DlGdrive:        kemonoDlGdrive,
				Configs:         kemonoConfig,
				SessionCookieId: kemonoSession,
				GdriveClient:    gdriveClient,
			}
			if kemonoCookieFile != "" {
				cookies, err := utils.ParseNetscapeCookieFile(
					kemonoCookieFile,
					kemonoSession,
					utils.KEMONO,
				)
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
						utils.ERROR,
					)
				}
				kemonoDlOptions.SessionCookies = cookies
			}

			kemonoDlOptions.ValidateArgs(kemonoUserAgent)

			utils.PrintWarningMsg()
			kemono.KemonoDownloadProcess(
				kemonoConfig,
				kemonoDl,
				kemonoDlOptions,
				kemonoDlFav,
			)
		},
	}
)

func init() {
	mutlipleUrlsMsg := "Multiple URLs can be supplied by separating them with a comma.\n" + 
						"Example: \"https://kemono.party/service/user/123,https://kemono.party/service/user/456\" (without the quotes)"
	kemonoCmd.Flags().StringVarP(
		&kemonoSession,
		"session",
		"s",
		"",
		utils.CombineStringsWithNewline(
			"Your Kemono Party \"session\" cookie value to use for the requests to Kemono Party.",
			"Required to get pass Kemono Party's DDOS protection and to download from your favourites.",
		),
	)
	kemonoCmd.MarkFlagRequired("session")
	kemonoCmd.Flags().StringSliceVar(
		&kemonoCreatorUrls,
		"creator_url",
		[]string{},
		utils.CombineStringsWithNewline(
			"Kemono Party creator URL(s) to download from.",
			mutlipleUrlsMsg,
		),
	)
	kemonoCmd.Flags().StringSliceVar(
		&kemonoPageNums,
		"page_num",
		[]string{},
		utils.CombineStringsWithNewline(
			"Min and max page numbers to search for corresponding to the order of the supplied Kemono Party creator URL(s).",
			"Format: \"num\", \"minNum-maxNum\", or \"\" to download all pages",
			"Leave blank to download all pages from each creator on Kemono Party.",
		),
	)
	kemonoCmd.Flags().StringSliceVar(
		&kemonoPostUrls,
		"post_url",
		[]string{},
		utils.CombineStringsWithNewline(
			"Kemono Party post URL(s) to download.",
			mutlipleUrlsMsg,
		),
	)
	kemonoCmd.Flags().BoolVarP(
		&kemonoDlGdrive,
		"dl_gdrive",
		"g",
		true,
		"Whether to download the Google Drive links of a post on Kemono Party.",
	)
	kemonoCmd.Flags().BoolVarP(
		&kemonoDlAttachments,
		"dl_attachments",
		"a",
		true,
		"Whether to download the attachments (images, zipped files, etc.) of a post on Kemono Party.",
	)
}
