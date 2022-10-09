package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv_fanbox"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

// Start the download process for Fantia
func FantiaDownloadProcess(
	fantiaPostIds, fanclubIds []string, cookies []http.Cookie,
	downloadThumbnail, downloadImages, downloadAttachments bool,
) {
	if !downloadThumbnail && !downloadImages && !downloadAttachments {
		return
	}

	var urlsToDownload []map[string]string
	if len(fantiaPostIds) > 0 {
		urlsArr := fantia.GetPostDetails(
			fantiaPostIds, cookies, 
			downloadThumbnail, downloadImages, downloadAttachments,
		)
		urlsToDownload = append(urlsToDownload, urlsArr...)
	}
	if len(fanclubIds) > 0 {
		fantiaPostIds := fantia.GetCreatorsPosts(fanclubIds, cookies)
		urlsArr := fantia.GetPostDetails(
			fantiaPostIds, cookies,
			downloadThumbnail, downloadImages, downloadAttachments,
		)
		urlsToDownload = append(urlsToDownload, urlsArr...)
	}

	if len(urlsToDownload) > 0 {
		request.DownloadURLsParallel(urlsToDownload, utils.MAX_CONCURRENT_DOWNLOADS, cookies, nil, nil)
	}
}

// Start the download process for Pixiv Fanbox
func PixivFanboxDownloadProcess(
	pixivFanboxPostIds, creatorIds []string, cookies []http.Cookie, gdriveApiKey string, gdrive *gdrive.GDrive,
	downloadThumbnail, downloadImages, downloadAttachments, downloadGdrive bool,
) {
	if !downloadThumbnail && !downloadImages && !downloadAttachments && !downloadGdrive {
		return
	}

	var urlsToDownload, gdriveUrlsToDownload []map[string]string
	if len(pixivFanboxPostIds) > 0 {
		urlsArr, gdriveArr := pixiv_fanbox.GetPostDetails(
			pixivFanboxPostIds, cookies,
			downloadThumbnail, downloadImages, downloadAttachments, downloadGdrive,
		)
		urlsToDownload = append(urlsToDownload, urlsArr...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveArr...)
	}
	if len(creatorIds) > 0 {
		fanboxIds := pixiv_fanbox.GetCreatorsPosts(creatorIds, cookies)
		urlsArr, gdriveArr := pixiv_fanbox.GetPostDetails(
			fanboxIds, cookies,
			downloadThumbnail, downloadImages, downloadAttachments, downloadGdrive,
		)
		urlsToDownload = append(urlsToDownload, urlsArr...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveArr...)
	}

	if len(urlsToDownload) > 0 {
		request.DownloadURLsParallel(
			urlsToDownload, utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
			cookies, pixiv_fanbox.GetPixivFanboxHeaders(), nil,
		)
	}
	if gdriveApiKey != "" && len(gdriveUrlsToDownload) > 0 {
		gdrive.DownloadGdriveUrls(gdriveUrlsToDownload)
	}
}

// Start the download process for Pixiv
func PixivDownloadProcess(
	artworkIds, illustratorIds, tagNames, pageNums []string,
	sortOrder, searchMode, ratingMode, artworkType, ugoiraOutputFormat, ffmpegPath, pixivRefreshToken string,
	deleteUgoiraZip bool, ugoiraQuality int, cookies []http.Cookie, pixivMobile *pixiv.PixivMobile,
) {
	// check if FFmpeg is installed
	cmd := exec.Command(ffmpegPath, "-version")
	err := cmd.Run()
	if err != nil {
		color.Red("FFmpeg is not installed. Please install FFmpeg and use the ffmpeg_path flag or add it to your PATH.")
		os.Exit(1)
	}

	var ugoiraToDownload []pixiv.Ugoira
	var artworksToDownload []map[string]string
	if len(artworkIds) > 0 {
		var artworksArr []map[string]string
		var ugoiraArr []pixiv.Ugoira
		if pixivRefreshToken == "" {
			artworksArr, ugoiraArr = pixiv.GetMultipleArtworkDetails(
				artworkIds, utils.DOWNLOAD_PATH, cookies,
			)
		} else {
			artworksArr, ugoiraArr = pixivMobile.GetMultipleArtworkDetails(
				artworkIds, utils.DOWNLOAD_PATH,
			)
		}
		artworksToDownload = append(artworksToDownload, artworksArr...)
		ugoiraToDownload = append(ugoiraToDownload, ugoiraArr...)
	}
	if len(illustratorIds) > 0 {
		var artworksArr []map[string]string
		var ugoiraArr []pixiv.Ugoira
		if pixivRefreshToken == "" {
			artworksArr, ugoiraArr = pixiv.GetMultipleIllustratorPosts(
				illustratorIds, utils.DOWNLOAD_PATH, artworkType, cookies,
			)
		} else {
			artworksArr, ugoiraArr = pixivMobile.GetMultipleIllustratorPosts(
				illustratorIds, utils.DOWNLOAD_PATH, artworkType,
			)
		}
		artworksToDownload = append(artworksToDownload, artworksArr...)
		ugoiraToDownload = append(ugoiraToDownload, ugoiraArr...)
	}
	if len(tagNames) > 0 {
		// loop through each tag and page number
		bar := utils.GetProgressBar(
			len(tagNames), 
			"Searching for artworks based on tag names...",
			utils.GetCompletionFunc(fmt.Sprintf("Finished searching for artworks based on %d tag names!", len(tagNames))),
		)
		for idx, tagName := range tagNames {
			var minPage, maxPage int
			if strings.Contains(pageNums[idx], "-") {
				tagPageNums := strings.SplitN(pageNums[idx], "-", 2)
				minPage, _ = strconv.Atoi(tagPageNums[0])
				maxPage, _ = strconv.Atoi(tagPageNums[1])
			} else {
				minPage, _ = strconv.Atoi(pageNums[idx])
				maxPage = minPage
			}
			var artworksArr []map[string]string
			var ugoiraArr []pixiv.Ugoira
			if pixivRefreshToken == "" {
				artworksArr, ugoiraArr = pixiv.TagSearch(
					tagName, utils.DOWNLOAD_PATH, sortOrder, searchMode, ratingMode, artworkType, minPage, maxPage, cookies,
				)
			} else {
				artworksArr, ugoiraArr = pixivMobile.TagSearch(
					tagName, searchMode, sortOrder, utils.DOWNLOAD_PATH, minPage, maxPage,
				)
			}
			artworksToDownload = append(artworksToDownload, artworksArr...)
			ugoiraToDownload = append(ugoiraToDownload, ugoiraArr...)
			bar.Add(1)
		}
	}

	if len(artworksToDownload) > 0 {
		request.DownloadURLsParallel(artworksToDownload, utils.PIXIV_MAX_CONCURRENT_DOWNLOADS, cookies, pixiv.GetPixivRequestHeaders(), nil)
	}
	if len(ugoiraToDownload) > 0 {
		pixiv.DownloadMultipleUgoira(ugoiraToDownload, ugoiraOutputFormat, ffmpegPath, deleteUgoiraZip, ugoiraQuality, cookies)
	}
}

// Main program
func main() {
	mutlipleIdsMsg := "For multiple IDs, separate them with a space.\nExample: \"12345 67891\""
	// Fantia args
	fantiaSession := flag.String(
		"fantia_session",
		"",
		"Your _session_id cookie value to use for the requests to Fantia.",
	)
	fanclub := flag.String(
		"fanclub_id",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Fantia Fanclub ID(s) to download from.",
				mutlipleIdsMsg,
			},
		),
	)
	fantiaPost := flag.String(
		"fantia_post",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Fantia post ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)

	// Pixiv Fanbox args
	pixivFanboxSession := flag.String(
		"fanbox_session",
		"",
		"Your FANBOXSESSID cookie value to use for the requests to Pixiv Fanbox.",
	)
	creator := flag.String(
		"creator_id",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Pixiv Fanbox Creator ID(s) to download from.",
				mutlipleIdsMsg,
			},
		),
	)
	pixivFanboxPost := flag.String(
		"fanbox_post",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Pixiv Fanbox post ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)

	// Fantia and Pixiv Fanbox args
	downloadThumbnail := flag.Bool(
		"download_thumbnail",
		true,
		"Whether to download the thumbnail of a post.",
	)
	downloadImages := flag.Bool(
		"download_images",
		true,
		"Whether to download the images of a post.",
	)
	downloadAttachments := flag.Bool(
		"download_attachments",
		true,
		"Whether to download the attachments of a post.",
	)
	downloadGdrive := flag.Bool(
		"download_gdrive",
		true,
		"Whether to download the Google Drive links of a Pixiv Fanbox post.",
	)

	// Pixiv args
	pixivStartOauth := flag.Bool(
		"pixiv_start_oauth",
		false,
		"Whether to start the Pixiv OAuth process to get one's refresh token.",
	)
	pixivRefreshToken := flag.String(
		"pixiv_refresh_token",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Your Pixiv refresh token to use for the requests to Pixiv.",
				"",
				"If you're downloading from Pixiv, it is recommended to use this flag",
				"instead of the \"-pixiv_session\" flag as there will be significantly lesser API calls to Pixiv.",
				"However, if you prefer more flexibility with your Pixiv downloads, you can use",
				"the \"-pixiv_session\" flag instead at the expense of longer API call time due to Pixiv's rate limiting.",
				"",
				"Note that you can get your refresh token by running the program with the \"-pixiv_start_oauth\" flag.",
			},
		),
	)
	pixivSession := flag.String(
		"pixiv_session",
		"",
		"Your PHPSESSID cookie value to use for the requests to Pixiv.",
	)
	deleteUgoiraZip := flag.Bool(
		"delete_ugoira_zip",
		true,
		"Whether to delete the downloaded ugoira zip file after conversion.",
	)
	ugoiraQuality := flag.Int(
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
				"",
				"For more information, see:",
				"- mp4: https://trac.ffmpeg.org/wiki/Encode/H.264#crf",
				"- webm: https://trac.ffmpeg.org/wiki/Encode/VP9#constantq\n",
			},
		),
	)
	ugoiraOutputFormat := flag.String(
		"ugoira_output_format",
		".gif",
		utils.CombineStringsWithNewline(
			[]string{
				"Output format for the ugoira conversion using FFmpeg.",
				fmt.Sprintf(
					"Accepted Extensions: %s\n",
					strings.TrimSpace(strings.Join(utils.UGOIRA_ACCEPTED_EXT, ", ")),
				),
			},
		),
	)
	artworkId := flag.String(
		"artwork_id",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Artwork ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)
	illustratorId := flag.String(
		"illustrator_id",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Illustrator ID(s) to download.",
				mutlipleIdsMsg,
			},
		),
	)
	tagName := flag.String(
		"tag_name",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Tag names to search for and download related artworks.",
				"For multiple tags, separate them with a comma.",
				"Example: \"tag name 1, tagName2\"",
			},
		),
	)
	pageNum := flag.String(
		"page_num",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied tag names.",
				"Format: \"pageNum\" or \"min-max\"",
				"Example: \"1\" or \"1-10\"",
			},
		),
	)
	sortOrder := flag.String(
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
	searchMode := flag.String(
		"search_mode",
		"s_tag_full",
		utils.CombineStringsWithNewline(
			[]string{
				"Search Mode Options:",
				"- s_tag: Match any post with SIMILAR tag name",
				"- s_tag_full: Match any post with the SAME tag name",
				"- s_tc: Match any post related by its title or caption",
				"Note that you can only specify ONE search mode per run!\n",
			},
		),
	)
	ratingMode := flag.String(
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
				"",
			},
		),
	)
	artworkType := flag.String(
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

	// Other args
	gdriveApiKey := flag.String(
		"gdrive_api_key",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Google Drive API key to use for downloading gdrive files.",
				"Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md",
			},
		),
	)
	downloadPath := flag.String(
		"download_path",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Configure the path to download the files to and save it for future runs.",
				"Note:",
				"If you had used the \"-download_path\" flag before or",
				"had used the Cultured Downloader software, you can leave this argument empty.",
			},
		),
	)
	ffmpegPath := flag.String(
		"ffmpeg_path",
		"ffmpeg",
		utils.CombineStringsWithNewline(
			[]string{
				"Configure the path to the FFmpeg executable.",
				"Download Link: https://ffmpeg.org/download.html\n",
			},
		),
	)
	version := flag.Bool(
		"version",
		false,
		"Display the current version of the Cultured Downloader CLI software.",
	)
	help := flag.Bool(
		"help",
		false,
		"Show the list of arguments with its description.",
	)
	flag.Parse()

	if *help || len(os.Args) == 1 {
		flag.PrintDefaults()
		return
	}
	if *version {
		fmt.Printf("Cultured Downloader CLI v%s by KJHJason\n", utils.VERSION)
		fmt.Println("GitHub Repo: https://github.com/KJHJason/Cultured-Downloader-CLI")
		return
	}
	if *pixivStartOauth {
		pixiv.NewPixivMobile("", 10).StartOauthFlow()
	}

	// check Pixiv args
	var pixivMobile *pixiv.PixivMobile
	if *pixivRefreshToken != "" {
		pixivMobile = pixiv.NewPixivMobile(*pixivRefreshToken, 10)
	}
	ugoiraOutputFormat = utils.CheckStrArgs(utils.UGOIRA_ACCEPTED_EXT, *ugoiraOutputFormat, "ugoira output format")
	sortOrder = utils.CheckStrArgs(utils.ACCEPTED_SORT_ORDER, *sortOrder, "sort order")
	searchMode = utils.CheckStrArgs(utils.ACCEPTED_SEARCH_MODE, *searchMode, "search mode")
	ratingMode = utils.CheckStrArgs(utils.ACCEPTED_RATING_MODE, *ratingMode, "rating mode")
	artworkType = utils.CheckStrArgs(utils.ACCEPTED_ARTWORK_TYPE, *artworkType, "artwork type")

	// Get the GDrive object
	var gdriveObj *gdrive.GDrive
	if *gdriveApiKey != "" {
		gdriveObj = gdrive.GetNewGDrive(*gdriveApiKey, utils.MAX_CONCURRENT_DOWNLOADS)
	}

	if *downloadPath != "" {
		utils.SetDefaultDownloadPath(*downloadPath)
		color.Green("Download path set to: %s", *downloadPath)
		return
	}
	if utils.DOWNLOAD_PATH == "" {
		color.Red(
			"Default download setting not found or is invalid, " +
				"please set up a default download path before continuing by pasing the -download_path flag.",
		)
		os.Exit(1)
	}

	// parse and verify the cookies
	fantiaCookie := api.VerifyAndGetCookie(api.Fantia, api.FantiaTitle, *fantiaSession)
	pixivFanboxCookie := api.VerifyAndGetCookie(api.PixivFanbox, api.PixivFanboxTitle, *pixivFanboxSession)
	pixivCookie := api.VerifyAndGetCookie(api.Pixiv, api.Pixiv, *pixivSession)
	cookies := []http.Cookie{fantiaCookie, pixivFanboxCookie, pixivCookie}

	// parse the ID(s) to download from
	fanclubIds := utils.SplitAndCheckIds(*fanclub)
	fantiaPostIds := utils.SplitAndCheckIds(*fantiaPost)
	creatorIds := utils.SplitArgs(*creator)
	pixivFanboxPostIds := utils.SplitAndCheckIds(*pixivFanboxPost)
	artworkIds := utils.SplitAndCheckIds(*artworkId)
	illustratorIds := utils.SplitAndCheckIds(*illustratorId)
	tagNames := utils.SplitArgsWithSep(*tagName, ",")
	pageNums := utils.SplitArgs(*pageNum)

	if len(tagNames) != len(pageNums) {
		color.Red("Number of tag names and page numbers must be equal.")
		os.Exit(1)
	}
	// check page nums if they are in the correct format
	pageNumsRegex := regexp.MustCompile(`^[1-9]\d*(-[1-9]\d*)?$`)
	for _, pageNum := range pageNums {
		if !pageNumsRegex.MatchString(pageNum) {
			color.Red("Invalid page number format: %s", pageNum)
			color.Red("Please follow the format, \"1-10\", as an example.")
			color.Red("Note that \"0\" are not accepted! E.g. \"0-9\" is invalid.")
			os.Exit(1)
		}
	}

	color.Yellow("CAUTION:")
	color.Yellow("Please do NOT stop the program while it is downloading.")
	color.Yellow("Doing so may result in incomplete downloads and corrupted files.")
	fmt.Println()
	FantiaDownloadProcess(
		fantiaPostIds, fanclubIds, cookies,
		*downloadThumbnail, *downloadImages, *downloadAttachments,
	)
	PixivFanboxDownloadProcess(
		pixivFanboxPostIds, creatorIds, cookies, *gdriveApiKey, gdriveObj,
		*downloadThumbnail, *downloadImages, *downloadAttachments, *downloadGdrive,
	)
	PixivDownloadProcess(
		artworkIds, illustratorIds, tagNames, pageNums,
		*sortOrder, *searchMode, *ratingMode, *artworkType, *ugoiraOutputFormat, *ffmpegPath,
		*pixivRefreshToken, *deleteUgoiraZip, *ugoiraQuality, cookies, pixivMobile,
	)
}
