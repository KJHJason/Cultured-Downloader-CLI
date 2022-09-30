package main

import (
	"os"
	"fmt"
	"flag"
	//"net/http"
	"cultured_downloader/utils"
)

func main() {
	fantia_session := flag.String("fantia_session", "", "Fantia session id to use.")
	fanclub := flag.String("fanclub", "", "Fanclub ids to download from.")
	fantia_post := flag.String("fantia_post", "", "Fantia post id to download.")

	pixiv_fanbox_session := flag.String("pixiv_fanbox_session", "", "Pixiv Fanbox session id to use.")
	creator := flag.String("creator", "", "Creator ids to download from.")
	pixiv_fanbox_post := flag.String("pixiv_fanbox_post", "", "Pixiv Fanbox post id to download.")

	downloadPath := flag.String("download_path", "", "Configure the path to download the files to.")
	help := flag.Bool("help", false, "Show help.")
	flag.Parse()

	if (*help) {
		flag.PrintDefaults()
		return
	}
	if (*downloadPath != "") {
		utils.SetDefaultDownloadPath(*downloadPath)
		fmt.Println("Download path has been changed to", *downloadPath)
		return
	}
	if (utils.GetDefaultDownloadPath() == "") {
		fmt.Println(
			"Default download setting not found or is invalid,",
			"please set up a default download path before continuing by pasing the -download_path flag.",
		)
		os.Exit(1)
	}

	// parse cookies
	fantia_cookie := utils.GetCookie(*fantia_session, "fantia")
	pixiv_fanbox_cookie := utils.GetCookie(*pixiv_fanbox_session, "fanbox")

	// verify cookies
	fantia_cookie_valid, err := utils.VerifyCookie(fantia_cookie, "fantia")
	if (err != nil) {
		utils.LogError(err, "", true)
	}
	if (*fantia_session != "" && !fantia_cookie_valid) {
		fmt.Println("Fantia cookie is invalid.")
		os.Exit(1)
	}

	pixiv_fanbox_cookie_valid, err := utils.VerifyCookie(pixiv_fanbox_cookie, "fanbox")
	if (err != nil) {
		utils.LogError(err, "", true)
	}
	if (*pixiv_fanbox_session != "" && !pixiv_fanbox_cookie_valid) {
		fmt.Println("Pixiv Fanbox cookie is invalid.")
		os.Exit(1)
	}

	// parse the ID(s) to download from
	fanclubIDs := utils.SplitArgs(*fanclub)
	fantiaPostIDs := utils.SplitArgs(*fantia_post)
	creatorIDs := utils.SplitArgs(*creator)
	pixivFanboxPostIDs := utils.SplitArgs(*pixiv_fanbox_post)
	fmt.Println("fanclubIDs", fanclubIDs)
	fmt.Println("fantiaPostIDs", fantiaPostIDs)
	fmt.Println("creatorIDs", creatorIDs)
	fmt.Println("pixivFanboxPostIDs", pixivFanboxPostIDs)

	// download
	// var urls_arr []map[string]string
	// url := map[string]string {
	// 	"url": "https://fantia.jp/posts/1132038/download/1810523",
	// 	"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	// }
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

	// // make a list of maps
	// utils.DownloadURLsParallel(urls_arr, []http.Cookie{fantia_cookie, pixiv_fanbox_cookie})
}