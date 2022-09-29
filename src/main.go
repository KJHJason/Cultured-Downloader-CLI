package main

import (
	"fmt"
	"flag"
	"strings"
	"net/http"
	"cultured_downloader/utils"
)

func main() {
	fantia_session := flag.String("fantia_session", "", "Fantia session id to use.")
	pixiv_fanbox_session := flag.String("pixiv_fanbox_session", "", "Pixiv Fanbox session id to use.")
	urls := flag.String("urls", "", "URLs to download. For multiple URLs, you can pass in like \"val1 val2\".")
	help := flag.Bool("help", false, "Show help.")
	flag.Parse()

	if (*help) {
		flag.PrintDefaults()
		return
	}

	urls_list := strings.Split(*urls, " ")
	fmt.Println("URLs:", urls_list)

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
		return
	}

	pixiv_fanbox_cookie_valid, err := utils.VerifyCookie(pixiv_fanbox_cookie, "fanbox")
	if (err != nil) {
		utils.LogError(err, "", true)
	}
	if (*pixiv_fanbox_session != "" && !pixiv_fanbox_cookie_valid) {
		fmt.Println("Pixiv Fanbox cookie is invalid.")
		return
	}

	// download
	var urls_arr []map[string]string
	url := map[string]string {
		"url": "https://fantia.jp/posts/1132038/download/1810523",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	url = map[string]string {
		"url": "https://fantia.jp/posts/1321871/download/2143558",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)
	url = map[string]string {
		"url": "https://fantia.jp/posts/1321871/download/2143557",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)
	url = map[string]string {
		"url": "https://fantia.jp/posts/1321871/download/2143559",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)
	url = map[string]string {
		"url": "https://fantia.jp/posts/1321871/download/2143560",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)
	url = map[string]string {
		"url": "https://fantia.jp/posts/1321871/download/2143561",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)
	url = map[string]string {
		"url": "https://fantia.jp/posts/1321871/download/2143562",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)
	url = map[string]string {
		"url": "https://c.fantia.jp/uploads/post/file/1481729/93929c30-f486-4d01-851c-a0d90ac44222.png",
		"filepath": "E:\\Codes\\Github Projects\\Cultured-Downloader-CLI\\src",
	}
	urls_arr = append(urls_arr, url)

	// make a list of maps
	utils.DownloadURLsParallel(urls_arr, []http.Cookie{fantia_cookie, pixiv_fanbox_cookie})
}