package cmds

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func init() {
	mutlipleIdsMsg := "For multiple IDs, separate them with a comma.\nExample: \"12345,67891\" (without the quotes)"

	// ------------------------------ Fantia ------------------------------ //

	fantiaCmd.Flags().StringVar(
		&fantiaSession,
		"session",
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
				"Format: \"num\" or \"minNum-maxNum\"",
				"Example: \"1\" or \"1-10\"",
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

	// ------------------------------ Pixiv Fanbox ------------------------------ //

	pixivFanboxCmd.Flags().StringVar(
		&fanboxSession,
		"session",
		"",
		"Your FANBOXSESSID cookie value to use for the requests to Pixiv Fanbox.",
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
	pixivFanboxCmd.Flags().BoolVar(
		&fanboxDlThumbnails,
		"dl_thumbnails",
		true,
		"Whether to download the thumbnail of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().BoolVar(
		&fanboxDlImages,
		"dl_images",
		true,
		"Whether to download the images of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().BoolVar(
		&fanboxDlAttachments,
		"dl_attachments",
		true,
		"Whether to download the attachments of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().BoolVar(
		&fanboxDlGdrive,
		"dl_gdrive",
		true,
		"Whether to download the Google Drive links of a Pixiv Fanbox post.",
	)
	pixivFanboxCmd.Flags().StringVar(
		&fanboxGdriveApiKey,
		"gdrive_api_key",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Google Drive API key to use for downloading gdrive files.",
				"Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md",
			},
		),
	)

	// ------------------------------ Pixiv ------------------------------ //

	pixivCmd.Flags().StringVar(
		&pixivFfmpegPath,
		"ffmpeg_path",
		"ffmpeg",
		utils.CombineStringsWithNewline(
			[]string{
				"Configure the path to the FFmpeg executable.",
				"Download Link: https://ffmpeg.org/download.html",
			},
		),
	)
	pixivCmd.Flags().BoolVar(
		&pixivStartOauth,
		"start_oauth",
		false,
		"Whether to start the Pixiv OAuth process to get one's refresh token.",
	)
	pixivCmd.Flags().StringVar(
		&pixivRefreshToken,
		"refresh_token",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Your Pixiv refresh token to use for the requests to Pixiv.",
				"If you're downloading from Pixiv, it is recommended to use this flag",
				"instead of the \"-pixiv_session\" flag as there will be significantly lesser API calls to Pixiv.",
				"However, if you prefer more flexibility with your Pixiv downloads, you can use",
				"the \"-pixiv_session\" flag instead at the expense of longer API call time due to Pixiv's rate limiting.",
				"Note that you can get your refresh token by running the program with the \"-pixiv_start_oauth\" flag.",
			},
		),
	)
	pixivCmd.Flags().StringVar(
		&pixivSession,
		"session",
		"",
		"Your PHPSESSID cookie value to use for the requests to Pixiv.",
	)
	pixivCmd.Flags().BoolVar(
		&deleteUgoiraZip,
		"delete_ugoira_zip",
		true,
		"Whether to delete the downloaded ugoira zip file after conversion.",
	)
	pixivCmd.Flags().IntVar(
		&ugoiraQuality,
		"ugoira_quality",
		10,
		utils.CombineStringsWithNewline(
			[]string{
				"Configure the quality of the converted ugoira (Only for .mp4 and .webm).",
				"This argument will be used as the crf value for FFmpeg.",
				"The lower the value, the higher the quality.",
				"Accepted values:",
				"- mp4: 0-51",
				"- webm: 0-63",
				"For more information, see:",
				"- mp4: https://trac.ffmpeg.org/wiki/Encode/H.264#crf",
				"- webm: https://trac.ffmpeg.org/wiki/Encode/VP9#constantq",
			},
		),
	)
	pixivCmd.Flags().StringVar(
		&ugoiraOutputFormat,
		"ugoira_output_format",
		".gif",
		utils.CombineStringsWithNewline(
			[]string{
				"Output format for the ugoira conversion using FFmpeg.",
				fmt.Sprintf(
					"Accepted Extensions: %s\n",
					strings.TrimSpace(strings.Join(pixiv.UGOIRA_ACCEPTED_EXT, ", ")),
				),
			},
		),
	)
	pixivCmd.Flags().StringSliceVar(
		&pixivArtworkIds,
		"artwork_id",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Artwork ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)
	pixivCmd.Flags().StringSliceVar(
		&pixivIllustratorIds,
		"illustrator_id",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Illustrator ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)
	pixivCmd.Flags().StringSliceVar(
		&pixivTagNames,
		"tag_name",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Tag names to search for and download related artworks.",
				"For multiple tags, separate them with a comma.",
				"Example: \"tag name 1, tagName2\"",
			},
		),
	)
	pixivCmd.Flags().StringSliceVar(
		&pixivPageNums,
		"tag_page_num",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied tag names.",
				"Format: \"num\" or \"minNum-maxNum\"",
				"Example: \"1\" or \"1-10\"",
			},
		),
	)
	pixivCmd.Flags().StringVar(
		&pixivSortOrder,
		"sort_order",
		"date_d",
		utils.CombineStringsWithNewline(
			[]string{
				"Download Order Options: date, popular, popular_male, popular_female",
				"Additionally, you can add the \"_d\" suffix for a descending order.",
				"Example: \"popular_d\"",
				"Note:",
				"- If using the \"-pixiv_refresh_token\" flag, only \"date\", \"date_d\", \"popular_d\" are supported.",
				"- Pixiv Premium is needed in order to search by popularity. Otherwise, Pixiv's API will default to \"date_d\".",
				"- You can only specify ONE tag name per run!\n",
			},
		),
	)
	pixivCmd.Flags().StringVar(
		&pixivSearchMode,
		"search_mode",
		"s_tag_full",
		utils.CombineStringsWithNewline(
			[]string{
				"Search Mode Options:",
				"- s_tag: Match any post with SIMILAR tag name",
				"- s_tag_full: Match any post with the SAME tag name",
				"- s_tc: Match any post related by its title or caption",
				"Note that you can only specify ONE search mode per run!",
			},
		),
	)
	pixivCmd.Flags().StringVar(
		&pixivRatingMode,
		"rating_mode",
		"all",
		utils.CombineStringsWithNewline(
			[]string{
				"Rating Mode Options:",
				"- r18: Restrict downloads to R-18 artworks",
				"- safe: Restrict downloads to all ages artworks",
				"- all: Include both R-18 and all ages artworks",
				"Notes:",
				"- You can only specify ONE rating mode per run!",
				"- If you're using the \"-pixiv_refresh_token\" flag, only \"all\" is supported.",
			},
		),
	)
	pixivCmd.Flags().StringVar(
		&pixivArtworkType,
		"artwork_type",
		"all",
		utils.CombineStringsWithNewline(
			[]string{
				"Artwork Type Options:",
				"- illust_and_ugoira: Restrict downloads to illustrations and ugoira only",
				"- manga: Restrict downloads to manga only",
				"- all: Include both illustrations, ugoira, and manga artworks",
				"Notes:",
				"- You can only specify ONE artwork type per run!",
				"- If you're using the \"-pixiv_refresh_token\" flag and are downloading by tag names, only \"all\" is supported.",
			},
		),
	)

	// ------------------------------ Common Flags ------------------------------ //

	type commonFlags struct {
		cmd 		   *cobra.Command
		overwriteVar   *bool
		cookieFileVar  *string
	}
	commonCmdFlags := []commonFlags{
		{
			cmd: fantiaCmd,
			overwriteVar: &fantiaOverwrite,
			cookieFileVar: &fantiaCookieFile,
		},
		{
			cmd: pixivFanboxCmd,
			overwriteVar: &fanboxOverwriteFiles,
			cookieFileVar: &fanboxCookieFile,
		},
		{
			cmd: pixivCmd,
			overwriteVar: &pixivOverwrite,
			cookieFileVar: &pixivCookieFile,
		},
	}
	for _, cmdInfo := range commonCmdFlags {
		cmd := cmdInfo.cmd

		cmd.Flags().BoolVarP(
			cmdInfo.overwriteVar,
			"overwrite",
			"o",
			false,
			utils.CombineStringsWithNewline(
				[]string{
					"Regardless of this flag, the program will overwrite any",
					"incomplete downloaded file by verifying against the Content-Length header response,",
					"if the size of locally downloaded file does not",
					"match with the header response, the program will re-download the file.",
					"Otherwise, if the Content-Length header is not present in the response,",
					"the program will only skip the file if the file already exists.",
					"However, if there's a need to overwrite the skipped files, you can use this flag to do so.",
					"You should only use this for websites like Pixiv Fanbox",
					"which does not return a Content-Length header in their response.",
					"However, do note that if you have an anti-virus program running,",
					"it might detect this program as a ransomware due as it is overwriting multiple files at once.",
					"Thus, it is recommended to not use this flag if you have an",
					"anti-virus program unless the anti-virus software has been disabled.",
				},
			),
		)
		cmd.Flags().StringVar(
			cmdInfo.cookieFileVar,
			"cookie_file",
			"",
			utils.CombineStringsWithNewline(
				[]string{
					"Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.",
					"You can generate a cookie file by using the \"Get cookies.txt\" extension for your browser.",
				},
			),
		)
	}

	// ------------------------------ Root CMD ------------------------------ //

	RootCmd.Flags().StringVar(
		&downloadPath,
		"download_path",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Configure the path to download the files to and save it for future runs. Otherwise, the program will use the current working directory.",
				"Note:",
				"If you had used the \"-download_path\" flag before or",
				"had used the Cultured Downloader Python program, the program will automatically use the path you had set.",
			},
		),
	)
	RootCmd.CompletionOptions.HiddenDefaultCmd = true
	RootCmd.AddCommand(
		fantiaCmd,
		pixivFanboxCmd,
		pixivCmd,
	)
}
