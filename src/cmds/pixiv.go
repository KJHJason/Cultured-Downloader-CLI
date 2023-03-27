package cmds

import (
	"fmt"
	"os"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/web"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/mobile"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/ugoira"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/cmds/textparser"
	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

var (
	pixivDlTextFile          string
	pixivCookieFile          string
	pixivFfmpegPath          string
	pixivStartOauth          bool
	pixivRefreshToken        string
	pixivSession             string
	deleteUgoiraZip          bool
	ugoiraQuality            int
	ugoiraOutputFormat       string
	pixivArtworkIds          []string
	pixivIllustratorIds      []string
	pixivIllustratorPageNums []string
	pixivTagNames            []string
	pixivPageNums            []string
	pixivSortOrder           string
	pixivSearchMode          string
	pixivRatingMode          string
	pixivArtworkType         string
	pixivOverwrite           bool
	pixivUserAgent           string
	pixivCmd                 = &cobra.Command{
		Use:   "pixiv",
		Short: "Download from Pixiv",
		Long:  "Supports downloads from Pixiv by artwork ID, illustrator ID, tag name, and more.",
		Run: func(cmd *cobra.Command, args []string) {
			request.CheckInternetConnection()

			if pixivStartOauth {
				err := pixivmobile.NewPixivMobile("", 10).StartOauthFlow()
				if err != nil {
					utils.LogError(
						err,
						"",
						true,
						utils.ERROR,
					)
				}
				return
			}

			pixivConfig := &configs.Config{
				FfmpegPath:     pixivFfmpegPath,
				OverwriteFiles: pixivOverwrite,
				UserAgent:      pixivUserAgent,
			}
			pixivConfig.ValidateFfmpeg()

			if pixivDlTextFile != "" {
				artworkIds, illustratorInfoSlice, tagInfoSlice := textparser.ParsePixivTextFile(pixivDlTextFile)
				pixivArtworkIds = append(pixivArtworkIds, artworkIds...)

				for _, illustratorInfo := range illustratorInfoSlice {
					pixivIllustratorIds = append(pixivIllustratorIds, illustratorInfo.ArtistId)
					pixivIllustratorPageNums = append(pixivIllustratorPageNums, illustratorInfo.PageNum)
				}

				for _, tagInfo := range tagInfoSlice {
					pixivTagNames = append(pixivTagNames, tagInfo.Tag)
					pixivPageNums = append(pixivPageNums, tagInfo.PageNum)
				}
			}
			pixivDl := &pixiv.PixivDl{
				ArtworkIds:          pixivArtworkIds,
				IllustratorIds:      pixivIllustratorIds,
				IllustratorPageNums: pixivIllustratorPageNums,
				TagNames:            pixivTagNames,
				TagNamesPageNums:    pixivPageNums,
			}
			pixivDl.ValidateArgs()

			pixivUgoiraOptions := &ugoira.UgoiraOptions{
				DeleteZip:    deleteUgoiraZip,
				Quality:      ugoiraQuality,
				OutputFormat: ugoiraOutputFormat,
			}
			pixivUgoiraOptions.ValidateArgs()

			if pixivRefreshToken == "" && pixivSession == "" {
				color.Red("You must provide a refresh token or session cookie ID to download from Pixiv.")
				os.Exit(1)
			}

			utils.PrintWarningMsg()
			if pixivRefreshToken != "" {
				pixivDlOptions := &pixivmobile.PixivMobileDlOptions{
					SortOrder:       pixivSortOrder,
					SearchMode:      pixivSearchMode,
					RatingMode:      pixivRatingMode,
					ArtworkType:     pixivArtworkType,
					Configs:         pixivConfig,
					RefreshToken:    pixivRefreshToken,
				}
				pixivDlOptions.ValidateArgs(pixivUserAgent)
				pixiv.PixivMobileDownloadProcess(
					pixivDl,
					pixivDlOptions,
					pixivUgoiraOptions,
				)
			} else {
				pixivDlOptions := &pixivweb.PixivWebDlOptions{
					SortOrder:       pixivSortOrder,
					SearchMode:      pixivSearchMode,
					RatingMode:      pixivRatingMode,
					ArtworkType:     pixivArtworkType,
					Configs:         pixivConfig,
					SessionCookieId: pixivSession,
				}
				if pixivCookieFile != "" {
					cookies, err := utils.ParseNetscapeCookieFile(
						pixivCookieFile,
						pixivSession,
						utils.PIXIV,
					)
					if err != nil {
						utils.LogError(
							err,
							"",
							true,
							utils.ERROR,
						)
					}
					pixivDlOptions.SessionCookies = cookies
				}
				pixivDlOptions.ValidateArgs(pixivUserAgent)
				pixiv.PixivWebDownloadProcess(
					pixivDl,
					pixivDlOptions,
					pixivUgoiraOptions,
				)
			}
		},
	}
)

func init() {
	mutlipleIdsMsg := getMultipleIdsMsg()
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
				"instead of the \"--session\" flag as there will be significantly lesser API calls to Pixiv.",
				"However, if you prefer more flexibility with your Pixiv downloads, you can use",
				"the \"--session\" flag instead at the expense of longer API call time due to Pixiv's rate limiting.",
				"Note that you can get your refresh token by running the program with the \"--start_oauth\" flag.",
			},
		),
	)
	pixivCmd.Flags().StringVarP(
		&pixivSession,
		"session",
		"s",
		"",
		"Your \"PHPSESSID\" cookie value to use for the requests to Pixiv.",
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
					strings.TrimSpace(strings.Join(ugoira.UGOIRA_ACCEPTED_EXT, ", ")),
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
		&pixivIllustratorPageNums,
		"illustrator_page_num",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied illustrator ID(s).",
				"Format: \"num\", \"minNum-maxNum\", or \"\" to download all pages",
				"Leave blank to download all pages from each illustrator.",
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
				"Min and max page numbers to search for corresponding to the order of the supplied tag name(s).",
				"Format: \"num\", \"minNum-maxNum\", or \"\" to download all pages",
				"Leave blank to search all pages for each tag name.",
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
				"- If using the \"--refresh_token\" flag, only \"date\", \"date_d\", \"popular_d\" are supported.",
				"- Pixiv Premium is needed in order to search by popularity. Otherwise, Pixiv's API will default to \"date_d\".",
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
				"- If you're using the \"--refresh_token\" flag, only \"all\" is supported.",
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
				"- If you're using the \"-pixiv_refresh_token\" flag and are downloading by tag names, only \"all\" is supported.",
			},
		),
	)
}
