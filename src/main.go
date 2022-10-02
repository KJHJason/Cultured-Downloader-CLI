package main

import (
	"os"
	"fmt"
	"flag"
	"regexp"
	"net/http"
	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func main() {
	fantia_session := flag.String("fantia_session", "", "Fantia session id to use.")
	fanclub := flag.String("fanclub", "", "Fanclub ids to download from.")
	fantia_post := flag.String("fantia_post", "", "Fantia post id to download.")

	pixiv_fanbox_session := flag.String("pixiv_fanbox_session", "", "Pixiv Fanbox session id to use.")
	creator := flag.String("creator", "", "Creator ids to download from.")
	pixiv_fanbox_post := flag.String("pixiv_fanbox_post", "", "Pixiv Fanbox post URL(s) to download.")

	gdrive_api_key := flag.String(
		"gdrive_api_key", 
		"", 
		"Google Drive API key to use for downloading gdrive files. " +
		"Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md",
	)
	downloadPath := flag.String("download_path", "", "Configure the path to download the files to.")
	help := flag.Bool("help", false, "Show help.")
	flag.Parse()

	if (*help) {
		flag.PrintDefaults()
		return
	}
	if (*downloadPath != "") {
		utils.SetDefaultDownloadPath(*downloadPath)
		color.Green("Download path set to: %s", *downloadPath)
		return
	}
	if (utils.DOWNLOAD_PATH == "") {
		color.Red(
			"Default download setting not found or is invalid, " +
			"please set up a default download path before continuing by pasing the -download_path flag.",
		)
		os.Exit(1)
	}

	// parse cookies
	fantia_cookie := utils.GetCookie(*fantia_session, "fantia")
	pixiv_fanbox_cookie := utils.GetCookie(*pixiv_fanbox_session, "fanbox")
	cookies := []http.Cookie{fantia_cookie, pixiv_fanbox_cookie}

	// verify cookies and gdrive api key
	fantia_cookie_valid, err := utils.VerifyCookie(fantia_cookie, "fantia")
	if (err != nil) {
		utils.LogError(err, "", true)
	}
	if (*fantia_session != "" && !fantia_cookie_valid) {
		color.Red("Fantia cookie is invalid.")
		os.Exit(1)
	}

	var gdrive *utils.GDrive
	if *gdrive_api_key != "" {
		gdrive = utils.GetNewGDrive(*gdrive_api_key, utils.MAX_CONCURRENT_DOWNLOADS)
	}

	pixiv_fanbox_cookie_valid, err := utils.VerifyCookie(pixiv_fanbox_cookie, "fanbox")
	if (err != nil) {
		utils.LogError(err, "", true)
	}
	if (*pixiv_fanbox_session != "" && !pixiv_fanbox_cookie_valid) {
		color.Red("Pixiv Fanbox cookie is invalid.")
		os.Exit(1)
	}

	// parse the ID(s) to download from
	fanclubIds := utils.SplitArgs(*fanclub)
	fantiaPostIds := utils.SplitArgs(*fantia_post)
	creatorIds := utils.SplitArgs(*creator)
	pixivFanboxPostUrls := utils.SplitArgs(*pixiv_fanbox_post)

	var urlsToDownload []map[string]string
	var gdriveUrlsToDownload []map[string]string
	if len(pixivFanboxPostUrls) > 0 {
		fanboxPostUrlRegex := regexp.MustCompile(
			`^https://(www\.fanbox\.cc/@[\w.-]+|[\w.-]+\.fanbox\.cc)/posts/\d+$`,
		) 
		for _, url := range pixivFanboxPostUrls {
			if !fanboxPostUrlRegex.MatchString(url) {
				color.Red("Invalid Pixiv Fanbox post URL: %s", url)
				os.Exit(1)
			}
		}
		urlsArr, gdriveArr := utils.GetPostDetails(pixivFanboxPostUrls, "fanbox", cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveArr...)
	}
	if len(creatorIds) > 0 {
		fanboxPostMap, _ := utils.GetCreatorsPosts(creatorIds, "fanbox", cookies)
		fanboxUrls := []string{}
		for _, post := range fanboxPostMap {
			url := fmt.Sprintf("https://www.fanbox.cc/@%s/posts/%s", post["creatorId"], post["postId"])
			fanboxUrls = append(fanboxUrls, url)
		}
		
		urlsArr, gdriveArr := utils.GetPostDetails(fanboxUrls, "fanbox", cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
		gdriveUrlsToDownload = append(gdriveUrlsToDownload, gdriveArr...)
	}
	if len(fantiaPostIds) > 0 {
		urlsArr, _ := utils.GetPostDetails(fantiaPostIds, "fantia", cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
	}
	if len(fanclubIds) > 0 {
		_, fantiaPostIds := utils.GetCreatorsPosts(fanclubIds, "fantia", cookies)
		urlsArr, _ := utils.GetPostDetails(fantiaPostIds, "fantia", cookies)
		urlsToDownload = append(urlsToDownload, urlsArr...)
	}

	fmt.Println(urlsToDownload)
	fmt.Println(gdriveUrlsToDownload)
	if *gdrive_api_key != "" {
		gdrive.DownloadGdriveUrls(gdriveUrlsToDownload)
	}
	// download
	// var urls_arr []map[string]string
	// url := map[string]string {
	// 	"url": "https://fantia.jp/posts/1132038/download/1810523",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://fantia.jp/posts/1321871/download/2143558",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://fantia.jp/posts/1321871/download/2143557",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://fantia.jp/posts/1321871/download/2143559",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://fantia.jp/posts/1321871/download/2143560",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://fantia.jp/posts/1321871/download/2143561",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://fantia.jp/posts/1321871/download/2143562",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)
	// url = map[string]string {
	// 	"url": "https://c.fantia.jp/uploads/post/file/1481729/93929c30-f486-4d01-851c-a0d90ac44222.png",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
	// urls_arr = append(urls_arr, url)

	// make a list of maps
	// utils.DownloadURLsParallel(urls_arr, []http.Cookie{fantia_cookie, pixiv_fanbox_cookie})
}