package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Start the download process for Fantia
func FantiaDownloadProcess(config *configs.Config, fantiaDl *fantia.FantiaDl, fantiaDlOptions *fantia.FantiaDlOptions) {
	if !fantiaDlOptions.DlThumbnails && !fantiaDlOptions.DlImages && !fantiaDlOptions.DlAttachments {
		return
	}

	var urlsToDownload []map[string]string
	if len(fantiaDl.PostIds) > 0 {
		urlsSlice := fantia.GetPostDetails(
			fantiaDl.PostIds,
			fantiaDlOptions,
		)
		urlsToDownload = append(urlsToDownload, urlsSlice...)
	}
	if len(fantiaDl.FanclubIds) > 0 {
		fantiaPostIds := fantia.GetCreatorsPosts(
			fantiaDl.FanclubIds,
			fantiaDlOptions.SessionCookies,
		)
		urlsSlice := fantia.GetPostDetails(
			fantiaPostIds,
			fantiaDlOptions,
		)
		urlsToDownload = append(urlsToDownload, urlsSlice...)
	}

	if len(urlsToDownload) > 0 {
		request.DownloadURLsParallel(
			urlsToDownload,
			utils.MAX_CONCURRENT_DOWNLOADS,
			fantiaDlOptions.SessionCookies,
			nil,
			nil,
			config.OverwriteFiles,
		)
	}
}

// Start the download process for Pixiv Fanbox
func PixivFanboxDownloadProcess(config *configs.Config, pixivFanboxDl *pixivfanbox.PixivFanboxDl, pixivFanboxDlOptions *pixivfanbox.PixivFanboxDlOptions) {
	if !pixivFanboxDlOptions.DlThumbnails && !pixivFanboxDlOptions.DlImages && !pixivFanboxDlOptions.DlAttachments && !pixivFanboxDlOptions.DlGdrive {
		return
	}

	var urlsToDownload, gdriveUrlsToDownload []map[string]string
	if len(pixivFanboxDl.PostIds) > 0 {
		urlsSlice, gdriveSlice := pixivfanbox.GetPostDetails(
			pixivFanboxDl.PostIds,
			pixivFanboxDlOptions,
		)
		urlsToDownload = append(urlsToDownload, urlsSlice...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveSlice...)
	}
	if len(pixivFanboxDl.CreatorIds) > 0 {
		fanboxIds := pixivfanbox.GetCreatorsPosts(
			pixivFanboxDl.CreatorIds,
			pixivFanboxDlOptions.SessionCookies,
		)
		urlsSlice, gdriveSlice := pixivfanbox.GetPostDetails(
			fanboxIds,
			pixivFanboxDlOptions,
		)
		urlsToDownload = append(urlsToDownload, urlsSlice...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveSlice...)
	}

	if len(urlsToDownload) > 0 {
		request.DownloadURLsParallel(
			urlsToDownload,
			utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
			pixivFanboxDlOptions.SessionCookies,
			pixivfanbox.GetPixivFanboxHeaders(),
			nil,
			config.OverwriteFiles,
		)
	}
	if config.GDriveClient != nil && len(gdriveUrlsToDownload) > 0 {
		config.GDriveClient.DownloadGdriveUrls(gdriveUrlsToDownload)
	}
}

// Start the download process for Pixiv
func PixivDownloadProcess(config *configs.Config, pixivDl *pixiv.PixivDl, pixivDlOptions *pixiv.PixivDlOptions, pixivUgoiraOptions *pixiv.UgoiraDlOptions) {
	var ugoiraToDownload []pixiv.Ugoira
	var artworksToDownload []map[string]string
	if len(pixivDl.ArtworkIds) > 0 {
		var artworksSlice []map[string]string
		var ugoiraSlice []pixiv.Ugoira
		if pixivDlOptions.MobileClient == nil {
			artworksSlice, ugoiraSlice = pixiv.GetMultipleArtworkDetails(
				pixivDl.ArtworkIds,
				utils.DOWNLOAD_PATH,
				pixivDlOptions.SessionCookies,
			)
		} else {
			artworksSlice, ugoiraSlice = pixivDlOptions.MobileClient.GetMultipleArtworkDetails(
				pixivDl.ArtworkIds,
				utils.DOWNLOAD_PATH,
			)
		}
		artworksToDownload = append(artworksToDownload, artworksSlice...)
		ugoiraToDownload = append(ugoiraToDownload, ugoiraSlice...)
	}

	if len(pixivDl.IllustratorIds) > 0 {
		var artworksSlice []map[string]string
		var ugoiraSlice []pixiv.Ugoira
		if pixivDlOptions.MobileClient == nil {
			artworksSlice, ugoiraSlice = pixiv.GetMultipleIllustratorPosts(
				pixivDl.IllustratorIds,
				utils.DOWNLOAD_PATH,
				pixivDlOptions.ArtworkType,
				pixivDlOptions.SessionCookies,
			)
		} else {
			artworksSlice, ugoiraSlice = pixivDlOptions.MobileClient.GetMultipleIllustratorPosts(
				pixivDl.IllustratorIds,
				utils.DOWNLOAD_PATH,
				pixivDlOptions.ArtworkType,
			)
		}
		artworksToDownload = append(artworksToDownload, artworksSlice...)
		ugoiraToDownload = append(ugoiraToDownload, ugoiraSlice...)
	}

	if len(pixivDl.TagNames) > 0 {
		// loop through each tag and page number
		bar := utils.GetProgressBar(
			len(pixivDl.TagNames),
			"Searching for artworks based on tag names...",
			utils.GetCompletionFunc(fmt.Sprintf("Finished searching for artworks based on %d tag names!", len(pixivDl.TagNames))),
		)
		for idx, tagName := range pixivDl.TagNames {
			var minPage, maxPage int
			pageNum := pixivDl.TagNamesPageNums[idx]
			if strings.Contains(pageNum, "-") {
				tagPageNums := strings.SplitN(pageNum, "-", 2)
				minPage, _ = strconv.Atoi(tagPageNums[0])
				maxPage, _ = strconv.Atoi(tagPageNums[1])
			} else {
				minPage, _ = strconv.Atoi(pageNum)
				maxPage = minPage
			}

			var artworksSlice []map[string]string
			var ugoiraSlice []pixiv.Ugoira
			if pixivDlOptions.MobileClient == nil {
				artworksSlice, ugoiraSlice = pixiv.TagSearch(
					tagName,
					utils.DOWNLOAD_PATH,
					pixivDlOptions,
					minPage,
					maxPage,
					pixivDlOptions.SessionCookies,
				)
			} else {
				artworksSlice, ugoiraSlice = pixivDlOptions.MobileClient.TagSearch(
					tagName,
					utils.DOWNLOAD_PATH,
					pixivDlOptions,
					minPage,
					maxPage,
				)
			}
			artworksToDownload = append(artworksToDownload, artworksSlice...)
			ugoiraToDownload = append(ugoiraToDownload, ugoiraSlice...)
			bar.Add(1)
		}
	}

	if len(artworksToDownload) > 0 {
		request.DownloadURLsParallel(
			artworksToDownload,
			utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
			pixivDlOptions.SessionCookies,
			pixiv.GetPixivRequestHeaders(),
			nil,
			config.OverwriteFiles,
		)
	}
	if len(ugoiraToDownload) > 0 {
		pixiv.DownloadMultipleUgoira(
			ugoiraToDownload,
			config,
			pixivUgoiraOptions,
			pixivDlOptions.SessionCookies,
		)
	}
}

var (
	downloadPath string
	rootCmd      = &cobra.Command{
		Use:     "cultured-downloader-cli",
		Version: fmt.Sprintf("%s by KJHJason\n%s", utils.VERSION, "GitHub Repo: https://github.com/KJHJason/Cultured-Downloader-CLI"),
		Short:   "Download images, videos, etc. from various websites like Fantia.",
		Long:    "Cultured Downloader CLI is a command-line tool for downloading images, videos, etc. from various websites like Pixiv, Pixiv Fanbox, and Fantia.",
		Run: func(cmd *cobra.Command, args []string) {
			if downloadPath != "" {
				err := utils.SetDefaultDownloadPath(downloadPath)
				if err != nil {
					color.Red(err.Error())
				} else {
					color.Green("Download path set to: %s", downloadPath)
				}
			}
		},
	}

	fantiaCookieFile    string
	fantiaSession       string
	fantiaFanclubIds    []string
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
			fantiaConfig := configs.Config{
				OverwriteFiles: fantiaOverwrite,
			}
			fantiaDl := fantia.FantiaDl{
				FanclubIds: fantiaFanclubIds,
				PostIds:    fantiaPostIds,
			}

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

			FantiaDownloadProcess(
				&fantiaConfig,
				&fantiaDl,
				&fantiaDlOptions,
			)
		},
	}

	fanboxCookieFile     string
	fanboxSession        string
	fanboxCreatorIds     []string
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
			pixivFanboxConfig := configs.Config{
				OverwriteFiles: fanboxOverwriteFiles,
			}
			if fanboxGdriveApiKey != "" {
				pixivFanboxConfig.GDriveClient = gdrive.GetNewGDrive(fanboxGdriveApiKey, utils.MAX_CONCURRENT_DOWNLOADS)
			}

			pixivFanboxDl := pixivfanbox.PixivFanboxDl{
				CreatorIds: fanboxCreatorIds,
				PostIds:    fanboxPostIds,
			}

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

			PixivFanboxDownloadProcess(
				&pixivFanboxConfig,
				&pixivFanboxDl,
				&pixivFanboxDlOptions,
			)
		},
	}

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

			pixivConfig := configs.Config{
				FfmpegPath:     pixivFfmpegPath,
				OverwriteFiles: pixivOverwrite,
			}
			pixivConfig.ValidateFfmpeg()

			utils.ValidatePageNumInput(
				len(pixivTagNames),
				pixivPageNums,
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

			pixivUgoiraOptions := pixiv.UgoiraDlOptions{
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

			PixivDownloadProcess(
				&pixivConfig,
				&pixivDl,
				&pixivDlOptions,
				&pixivUgoiraOptions,
			)
		},
	}
)

func init() {
	mutlipleIdsMsg := "For multiple IDs, separate them with a space.\nExample: \"12345 67891\""

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
		"tag_name_page_num",
		[]string{},
		utils.CombineStringsWithNewline(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied tag names.",
				"Format: \"pageNum\" or \"min-max\"",
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

	cmds := []*cobra.Command{
		fantiaCmd,
		pixivFanboxCmd,
		pixivCmd,
	}
	overwriteVar := []*bool{
		&fantiaOverwrite,
		&pixivOverwrite,
	}
	cookieFileVar := []*string{
		&fantiaCookieFile,
		&fanboxCookieFile,
		&pixivCookieFile,
	}

	overwriteIdx := 0
	for idx, cmd := range cmds {
		if cmd != pixivFanboxCmd {
			cmd.Flags().BoolVarP(
				overwriteVar[overwriteIdx],
				"overwrite",
				"o",
				false,
				utils.CombineStringsWithNewline(
					[]string{
						"Use this flag to overwrite any existing but incomplete downloaded files when downloading.",
						"Does this by verifying against the Content-Length header response,",
						"if the size of locally downloaded file does not match with the header response, overwrite the existing file.",
						"However, do note that if you many incomplete downloads and have an anti-virus program running,",
						"it might detect this program as a ransomware due as it is overwriting multiple files at once.",
						"Thus, it is recommended to not use this flag if you have an anti-virus program unless the anti-virus software has been disabled.",
					},
				),
			)
			overwriteIdx++
		}

		cmd.Flags().StringVar(
			cookieFileVar[idx],
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

	rootCmd.Flags().StringVar(
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
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.AddCommand(
		fantiaCmd,
		pixivFanboxCmd,
		pixivCmd,
	)
}

// Main program
func main() {
	rootCmd.Execute()
}
