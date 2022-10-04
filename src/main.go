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
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

func FantiaDownloadProcess(fantiaPostIds, fanclubIds []string, cookies []http.Cookie) {
	var urlsToDownload []map[string]string
	if len(fantiaPostIds) > 0 {
		urlsArr, _ := utils.GetPostDetails(fantiaPostIds, utils.Fantia, cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
	}
	if len(fanclubIds) > 0 {
		fantiaPostIds := utils.GetCreatorsPosts(fanclubIds, utils.Fantia, cookies)
		urlsArr, _ := utils.GetPostDetails(fantiaPostIds, utils.Fantia, cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
	}

	if len(urlsToDownload) > 0 {
		utils.DownloadURLsParallel(urlsToDownload, utils.MAX_CONCURRENT_DOWNLOADS, cookies, nil, nil)
	}
}

func PixivFanboxDownloadProcess(pixivFanboxPostIds, creatorIds []string, cookies []http.Cookie, gdriveApiKey string, gdrive *utils.GDrive) {
	var urlsToDownload, gdriveUrlsToDownload []map[string]string
	if len(pixivFanboxPostIds) > 0 {
		urlsArr, gdriveArr := utils.GetPostDetails(pixivFanboxPostIds, utils.PixivFanbox, cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveArr...)
	}
	if len(creatorIds) > 0 {
		fanboxIds := utils.GetCreatorsPosts(creatorIds, utils.PixivFanbox, cookies)
		urlsArr, gdriveArr := utils.GetPostDetails(fanboxIds, utils.PixivFanbox, cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveArr...)
	}

	if len(urlsToDownload) > 0 {
		utils.DownloadURLsParallel(urlsToDownload, utils.PIXIV_MAX_CONCURRENT_DOWNLOADS, cookies, utils.GetPixivFanboxHeaders(), nil)
	}
	if gdriveApiKey != "" {
		gdrive.DownloadGdriveUrls(gdriveUrlsToDownload)
	}
}

func PixivDownloadProcess(
	artworkIds, illustratorIds, tagNames, pageNums []string, 
	sortOrder, searchMode, ratingMode, artworkType, ugoiraOutputFormat, ffmpegPath string, 
	deleteUgoiraZip bool, cookies []http.Cookie,
) {
	// check if FFmpeg is installed
	cmd := exec.Command(ffmpegPath, "-version")
	err := cmd.Run()
	if err != nil {
		color.Red("FFmpeg is not installed. Please install FFmpeg and use the ffmpeg_path flag or add it to your PATH.")
		os.Exit(1)
	}

	var ugoiraToDownload []utils.Ugoira
	var artworksToDownload []map[string]string
	if len(artworkIds) > 0 {
		artworksArr, ugoiraArr := utils.GetMultipleArtworkDetails(
			artworkIds, utils.DOWNLOAD_PATH, cookies,
		)
		artworksToDownload = append(artworksToDownload, artworksArr...)
		ugoiraToDownload = append(ugoiraToDownload, ugoiraArr...)
	}
	if len(illustratorIds) > 0 {
		artworksArr, ugoiraArr := utils.GetMultipleIllustratorPosts(
			illustratorIds, utils.DOWNLOAD_PATH, artworkType, cookies,
		)
		artworksToDownload = append(artworksToDownload, artworksArr...)
		ugoiraToDownload = append(ugoiraToDownload, ugoiraArr...)
	}
	if len(tagNames) > 0 {
		// loop through each tag and page number
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
			artworksArr, ugoiraArr := utils.TagSearch(
				tagName, utils.DOWNLOAD_PATH, sortOrder, searchMode, ratingMode, artworkType, minPage, maxPage, cookies,
			)
			artworksToDownload = append(artworksToDownload, artworksArr...)
			ugoiraToDownload = append(ugoiraToDownload, ugoiraArr...)
		}
	}

	if len(artworksToDownload) > 0 {
		utils.DownloadURLsParallel(artworksToDownload, utils.PIXIV_MAX_CONCURRENT_DOWNLOADS, cookies, utils.GetPixivRequestHeaders(), nil)
	}
	if len(ugoiraToDownload) > 0 {
		utils.DownloadUgoira(ugoiraToDownload, ugoiraOutputFormat, ffmpegPath, deleteUgoiraZip, cookies)
	}
}

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
		utils.CombineStrings(
			[]string{
				"Fantia Fanclub ID(s) to download from.",
				mutlipleIdsMsg,
			}, "\n",
		),
	)
	fantiaPost := flag.String(
		"fantia_post",
		"",
		utils.CombineStrings(
			[]string{
				"Fantia post ID(s) to download.",
				mutlipleIdsMsg,
			}, "\n",
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
		utils.CombineStrings(
			[]string{
				"Pixiv Fanbox Creator ID(s) to download from.",
				mutlipleIdsMsg,
			}, "\n",
		),
	)
	pixivFanboxPost := flag.String(
		"fanbox_post",
		"",
		utils.CombineStrings(
			[]string{
				"Pixiv Fanbox post ID(s) to download.",
				mutlipleIdsMsg,
			}, "\n",
		),
	)

	// Pixiv args
	pixivSession := flag.String(
		"pixiv_session",
		"",
		"Your PHPSESSID cookie value to use for the requests to Pixiv.",
	)
	deleteUgoiraZip := flag.Bool(
		"delete_ugoira_zip",
		false,
		"Whether to delete the downloaded ugoira zip file after conversion.",
	)
	ugoiraOutputFormat := flag.String(
		"ugoira_output_format",
		".gif",
		utils.CombineStrings(
			[]string{
				"Output format for the ugoira conversion using FFmpeg.",
				fmt.Sprintf(
					"Accepted Extensions: %s",
					strings.TrimSpace(strings.Join(utils.UGOIRA_ACCEPTED_EXT, ", ")),
				),
				"Note:",
				// TODO: Check if the notes are accurate
				"- .webm will take MORE time to convert and will have a LARGER file size but will have a BETTER quality.",
				"- .mp4 will take LESS time to convert with ACCEPTABLE quality and SMALLER file size.",
				"- .gif will take LESS time to convert with ACCEPTABLE quality but with a LARGER file size.\n",
			}, "\n",
		),
	)
	artworkId := flag.String(
		"artwork_id",
		"",
		utils.CombineStrings(
			[]string{
				"Artwork ID(s) to download.",
				mutlipleIdsMsg,
			}, "\n",
		),
	)
	illustratorId := flag.String(
		"illustrator_id",
		"",
		utils.CombineStrings(
			[]string{
				"Illustrator ID(s) to download.",
				mutlipleIdsMsg,
			}, "\n",
		),
	)
	tagName := flag.String(
		"tag_name",
		"",
		utils.CombineStrings(
			[]string{
				"Tag names to search for and download related artworks.",
				"For multiple tags, separate them with a comma.",
				"Example: \"tag name 1, tagName2\"",
			}, "\n",
		),
	)
	pageNum := flag.String(
		"page_num",
		"",
		utils.CombineStrings(
			[]string{
				"Min and max page numbers to search for corresponding to the order of the supplied tag names.",
				"Format: \"pageNum\" or \"min-max\"",
				"Example: \"1\" or \"1-10\"",
			}, "\n",
		),
	)
	sortOrder := flag.String(
		"sort_order",
		"date_d",
		utils.CombineStrings(
			[]string{
				"Download Order Options: date, popular, popular_male, popular_female",
				"Additionally, you can add the \"_d\" suffix for a descending order.",
				"Example: \"popular_d\"",
				"Note that you can only specify ONE tag name per run!\n",
			}, "\n",
		),
	)
	searchMode := flag.String(
		"search_mode",
		"s_tag_full",
		utils.CombineStrings(
			[]string{
				"Search Mode Options:",
				"- s_tag: Match any post with SIMILAR tag name",
				"- s_tag_full: Match any post with the SAME tag name",
				"- s_tc: Match any post related by its title or caption",
				"Note that you can only specify ONE search mode per run!\n",
			}, "\n",
		),
	)
	ratingMode := flag.String(
		"rating_mode",
		"all",
		utils.CombineStrings(
			[]string{
				"Rating Mode Options:",
				"- r18: Restrict downloads to R-18 artworks",
				"- safe: Restrict downloads to all ages artworks",
				"- all: Include both R-18 and all ages artworks",
				"Note that you can only specify ONE rating mode per run!\n",
			}, "\n",
		),
	)
	artworkType := flag.String(
		"artwork_type",
		"illust_and_ugoira",
		utils.CombineStrings(
			[]string{
				"Artwork Type Options:",
				"- illust_and_ugoira: Restrict downloads to illustrations and ugoira only",
				"- manga: Restrict downloads to manga only",
				"- all: Include both illustrations, ugoira, and manga artworks",
				"Note that you can only specify ONE artwork type per run!",
			}, "\n",
		),
	)

	// Other args
	gdriveApiKey := flag.String(
		"gdrive_api_key",
		"",
		utils.CombineStrings(
			[]string{
				"Google Drive API key to use for downloading gdrive files.",
				"Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md",
			}, "\n",
		),
	)
	downloadPath := flag.String(
		"download_path", 
		"",
		utils.CombineStrings(
			[]string{
				"Configure the path to download the files to and save it for future runs.",
				"Note:",
				"If you had used the \"-download_path\" flag before or",
				"had used the Cultured Downloader software, you can leave this argument empty.",
			}, "\n",
		),
	)
	ffmpegPath := flag.String(
		"ffmpeg_path", 
		"ffmpeg", 
		utils.CombineStrings(
			[]string{
				"Configure the path to the FFmpeg executable.",
				"Download Link: https://ffmpeg.org/download.html\n",
			}, "\n",
		),
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

	// check ugoira output format
	ugoiraExtIsValid := false
	for _, format := range utils.UGOIRA_ACCEPTED_EXT {
		if *ugoiraOutputFormat == format {
			ugoiraExtIsValid = true
			break
		}
	}
	if !ugoiraExtIsValid {
		color.Red("Invalid ugoira output format: %s", *ugoiraOutputFormat)
		color.Red(
			fmt.Sprintf(
				"Valid ugoira output formats: %s",
				strings.TrimSpace(strings.Join(utils.UGOIRA_ACCEPTED_EXT, ", ")),
			),
		)
		os.Exit(1)
	}

	// Get the GDrive object
	var gdrive *utils.GDrive
	if *gdriveApiKey != "" {
		gdrive = utils.GetNewGDrive(*gdriveApiKey, utils.MAX_CONCURRENT_DOWNLOADS)
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
	fantiaCookie := utils.VerifyAndGetCookie(utils.Fantia, utils.FantiaTitle, *fantiaSession)
	pixivFanboxCookie := utils.VerifyAndGetCookie(utils.PixivFanbox, utils.PixivFanboxTitle, *pixivFanboxSession)
	pixivCookie := utils.VerifyAndGetCookie(utils.Pixiv, utils.Pixiv, *pixivSession)
	cookies := []http.Cookie{fantiaCookie, pixivFanboxCookie, pixivCookie}

	// parse the ID(s) to download from
	fanclubIds := utils.SplitArgs(*fanclub)
	fantiaPostIds := utils.SplitArgs(*fantiaPost)
	creatorIds := utils.SplitArgs(*creator)
	pixivFanboxPostIds := utils.SplitArgs(*pixivFanboxPost)
	artworkIds := utils.SplitArgs(*artworkId)
	illustratorIds := utils.SplitArgs(*illustratorId)
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

	FantiaDownloadProcess(fantiaPostIds, fanclubIds, cookies)
	PixivFanboxDownloadProcess(pixivFanboxPostIds, creatorIds, cookies, *gdriveApiKey, gdrive)
	PixivDownloadProcess(
		artworkIds, illustratorIds, tagNames, pageNums, 
		*sortOrder, *searchMode, *ratingMode, *artworkType, 
		*ugoiraOutputFormat, *ffmpegPath, *deleteUgoiraZip, cookies,
	)
}
