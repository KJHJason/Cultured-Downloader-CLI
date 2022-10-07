package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"reflect"
	"time"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

const (
	illust = iota
	manga
	ugoira
)

// This is due to Pixiv's strict rate limiting.
// 
// Without delays, the user might get 429 too many requests
// or the user's account might get suspended.
//
// Additionally, pixiv.net is protected by cloudflare, so
// to prevent the user's IP reputation from going down, delays are added.
//
// More info: https://github.com/Nandaka/PixivUtil2/issues/477
func PixivSleep() {
	time.Sleep(utils.GetRandomTime(1.5, 2.5))
}

// Returns a defined request header needed to communicate with Pixiv's API
func GetPixivRequestHeaders() map[string]string {
	return map[string]string{
		"Origin": "https://www.pixiv.net/",
		"Referer": "https://www.pixiv.net/",
	}
}

type Ugoira struct {
	Url string
	FilePath string
	Frames map[string]int64
}

// Process the artwork details JSON and returns a map of urls with its file path and a slice of Ugoira structures
func ProcessArtworkJson(artworkUrlsJson interface{}, artworkType int64, postDownloadDir string) ([]map[string]string, Ugoira) {
	if artworkUrlsJson == nil {
		return nil, Ugoira{}
	}

	var urlsToDownload []map[string]string
	var ugoiraDownloadInfo Ugoira
	artworkUrls := artworkUrlsJson.(map[string]interface{})["body"]
	if artworkType == ugoira {
		ugoiraMap := artworkUrls.(map[string]interface{})
		originalUrl := ugoiraMap["originalSrc"].(string)

		// map the files to their delay
		frameInfoMap := map[string]int64{}
		for _, frame := range ugoiraMap["frames"].([]interface{}) {
			frameMap := frame.(map[string]interface{})
			frameInfoMap[frameMap["file"].(string)] = int64(frameMap["delay"].(float64))
		}

		ugoiraDownloadInfo = Ugoira{
			Url: originalUrl,
			FilePath: postDownloadDir,
			Frames: frameInfoMap,
		}
	} else {
		for _, artworkUrl := range artworkUrls.([]interface{}) {
			artworkUrlMap := artworkUrl.(map[string]interface{})["urls"].(map[string]interface{})
			originalUrl := artworkUrlMap["original"].(string)
			urlsToDownload = append(urlsToDownload, map[string]string{
				"url": originalUrl,
				"filepath": postDownloadDir,
			})
		}
	}
	return urlsToDownload, ugoiraDownloadInfo
}

type ArtworkDetails struct {
	Body struct {
		UserName string `json:"userName"`
		Title string `json:"title"`
		IllustType int64 `json:"illustType"`
	}
}

// Retrieves details of an artwork ID and returns 
// the folder path to download the artwork to, the JSON response, and the artwork type
func GetArtworkDetails(artworkId, downloadPath string, cookies []http.Cookie) (string, interface{}, int64) {
	if artworkId == "" {
		return "", nil, -1
	}

	headers := GetPixivRequestHeaders()
	url := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", artworkId)
	artworkDetailsRes, err := request.CallRequest("GET", url, 15, cookies, headers, nil, false)
	if err != nil {
		errorMsg := fmt.Sprintf("Pixiv: Failed to get artwork details for %v\nURL: %s", artworkId, url)
		utils.LogError(err, errorMsg, false)
		return "", nil, -1
	}
	if artworkDetailsRes == nil {
		return "", nil, -1
	}
	defer artworkDetailsRes.Body.Close()
	if artworkDetailsRes.StatusCode != 200 {
		errorMsg := fmt.Sprintf(
			"Pixiv: Failed to get details for artwork ID %s due to %s response\nURL: %s",
			artworkId,
			artworkDetailsRes.Status,
			url,
		)
		utils.LogError(nil, errorMsg, false)
		return "", nil, -1
	}
	var artworkDetailsJsonRes ArtworkDetails
	resBody := utils.ReadResBody(artworkDetailsRes)
	err = json.Unmarshal(resBody, &artworkDetailsJsonRes)
	if err != nil {
		utils.LogError(
			err, 
			fmt.Sprintf("Pixiv: Failed to unmarshal artwork details for %s\nJSON: %s", artworkId, string(resBody)),
			false,
		)
		return "", nil, -1
	}
	artworkJsonBody := artworkDetailsJsonRes.Body
	illustratorName := artworkJsonBody.UserName
	artworkName := artworkJsonBody.Title
	artworkPostDir := utils.CreatePostFolder(filepath.Join(downloadPath, PixivTitle), illustratorName, artworkId, artworkName)

	artworkType := artworkJsonBody.IllustType
	switch artworkType {
	case illust, manga: // illustration or manga
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/pages", artworkId)
	case ugoira: // ugoira
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/ugoira_meta", artworkId)
	default:
		errorMsg := fmt.Sprintf("Pixiv: Unsupported artwork type %d for artwork ID %s", artworkType, artworkId)
		utils.LogError(nil, errorMsg, false)
		return "", nil, -1
	}

	PixivSleep()
	artworkUrlsRes, err := request.CallRequest("GET", url, 15, cookies, headers, nil, false)
	if err != nil {
		errorMsg := fmt.Sprintf("Pixiv: Failed to get artwork URLs for %s\nURL: %s", artworkId, url)
		utils.LogError(err, errorMsg, false)
		return "", nil, -1
	}
	defer artworkUrlsRes.Body.Close()
	if artworkUrlsRes.StatusCode != 200 {
		errorMsg := fmt.Sprintf(
			"Pixiv: Failed to get details for artwork URLs %v due to %s response\nURL: %s",
			artworkId,
			artworkUrlsRes.Status,
			url,
		)
		utils.LogError(nil, errorMsg, false)
		return "", nil, -1
	}
	return artworkPostDir, utils.LoadJsonFromResponse(artworkUrlsRes), artworkType
}

// Retrieves multiple artwork details based on the given slice of artwork IDs
// and returns a map to use for downloading and a slice of Ugoira structures
func GetMultipleArtworkDetails(artworkIds []string, downloadPath string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	bar := utils.GetProgressBar(
		len(artworkIds), 
		"Getting and processing artwork details...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting and processing %d artwork details!", len(artworkIds))),
	)
	var ugoiraDetails []Ugoira
	var artworkDetails []map[string]string
	lastArtworkId := artworkIds[len(artworkIds)-1]
	for _, artworkId := range artworkIds {
		artworkFolder, jsonRes, artworkType := GetArtworkDetails(artworkId, downloadPath, cookies)
		if artworkFolder == "" || artworkType == -1 {
			continue
		}
		processedJsonRes, ugoiraInfo := ProcessArtworkJson(jsonRes, artworkType, artworkFolder)
		if artworkType == ugoira {
			ugoiraDetails = append(ugoiraDetails, ugoiraInfo)
		} else {
			artworkDetails = append(artworkDetails, processedJsonRes...)
		}

		bar.Add(1)
		if artworkId != lastArtworkId {
			PixivSleep()
		}
	}

	return artworkDetails, ugoiraDetails
}

// Converts the Ugoira to the desired output path using FFmpeg
func ConvertUgoira(ugoiraInfo Ugoira, imagesFolderPath, outputPath string, ffmpegPath string, ugoiraQuality int) {
	outputExt := filepath.Ext(outputPath)
	if !utils.ArrContains(utils.UGOIRA_ACCEPTED_EXT, outputExt) {
		panic(fmt.Sprintf("Pixiv: Output extension %v is not allowed for ugoira conversion", outputExt))
	}

	// sort the ugoira frames by their filename which are %6d.imageExt
	sortedFilenames := make([]string, 0, len(ugoiraInfo.Frames))
	for fileName := range ugoiraInfo.Frames {
		sortedFilenames = append(sortedFilenames, fileName)
	}
	sort.Strings(sortedFilenames)

	// write the frames' variable delays to a text file
	baseFmtStr := "file '%s'\nduration %f\n"
	delaysText := "ffconcat version 1.0\n"
	for _, frameName := range sortedFilenames {
		delay := ugoiraInfo.Frames[frameName]
		delaySec := float64(delay) / 1000
		delaysText += fmt.Sprintf(
			baseFmtStr,
			filepath.Join(imagesFolderPath, frameName), delaySec,
		)
	}
	// copy the last frame and add it to the end of the delays text file
	// https://video.stackexchange.com/questions/20588/ffmpeg-flash-frames-last-still-image-in-concat-sequence
	lastFilename := sortedFilenames[len(sortedFilenames)-1]
	delaysText += fmt.Sprintf(
		"file '%s'\nduration %f", 
		filepath.Join(imagesFolderPath, lastFilename),
		0.001,
	)

	concatDelayFilePath := filepath.Join(imagesFolderPath, "delays.txt")
	f, err := os.Create(concatDelayFilePath)
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString(delaysText)
	if err != nil {
		f.Close()
		panic(err)
	}
	f.Close()

	// FFmpeg flags: https://www.ffmpeg.org/ffmpeg.html
	args := []string{
		"-y", 						// overwrite output file if it exists
		"-an",					 	// disable audio
		"-f", "concat", 			// input is a concat file
		"-safe", "0",   			// allow absolute paths in the concat file
		"-i", concatDelayFilePath,	// input file
	}
	if outputExt == ".webm" || outputExt == ".mp4" {
		// if converting the ugoira to a webm or .mp4 file 
		// then set the output video codec to vp9 or h264 respectively
		// 	- webm: https://trac.ffmpeg.org/wiki/Encode/VP9
		// 	- mp4: https://trac.ffmpeg.org/wiki/Encode/H.264
		encodingMap := map[string]string{
			".webm": "libvpx-vp9",
			".mp4": "libx264",
		}
		if outputExt == ".mp4" {
			// if converting to an mp4 file
			// crf range is 0-51 for .mp4 files
			if ugoiraQuality > 51 {
				ugoiraQuality = 51
			} else if ugoiraQuality < 0 {
				ugoiraQuality = 0
			}
			args = append(
				args,
				"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2", // pad the video to be even
				"-crf", strconv.Itoa(ugoiraQuality), // set the quality
			)
		} else {
			// crf range is 0-63 for .webm files
			if ugoiraQuality == 0 || ugoiraQuality < 0 {
				args = append(args, "-lossless", "1") 
			} else if ugoiraQuality > 63 {
				args = append(args, "-crf", "63")
			} else {
				args = append(args, "-crf", strconv.Itoa(ugoiraQuality))
			}
		}

		args = append(
			args,
			"-pix_fmt", "yuv420p", 			// set the pixel format to yuv420p
			"-c:v", encodingMap[outputExt], // video codec
			"-vsync", "passthrough", 		// Prevents frame dropping
		)
	} else if outputExt == ".gif" {
		// Generate a palette for the gif using FFmpeg for better quality
		palettePath := filepath.Join(imagesFolderPath, "palette.png")
		ffmpegImages := "%" + fmt.Sprintf(
			"%dd%s", // Usually it's %6d.extension but just in case, measure the length of the filename
			len(utils.RemoveExtFromFilename(sortedFilenames[0])), 
			filepath.Ext(sortedFilenames[0]),
		)
		imagePaletteCmd := exec.Command(
			ffmpegPath,
			"-i", filepath.Join(imagesFolderPath, ffmpegImages),
			"-vf", "palettegen",
			palettePath,
		)
		// imagePaletteCmd.Stderr = os.Stderr
		// imagePaletteCmd.Stdout = os.Stdout
		err = imagePaletteCmd.Run()
		if err != nil {
			panic("Pixiv: Failed to generate palette for ugoira gif")
		}
		args = append(
			args,
			"-loop", "0", // loop the gif
			"-i", palettePath,
			"-filter_complex", "paletteuse",
		)
	} else if outputExt == ".apng" {
		args = append(
			args,
			"-plays", "0", 								// loop the apng
			"-vf", 
			"setpts=PTS-STARTPTS,hqdn3d=1.5:1.5:6:6", 	// set the setpts filter and apply some denoising
		)
	} else { // outputExt == ".webp"
		args = append(
			args,
			"-pix_fmt", "yuv420p",   // set the pixel format to yuv420p
			"-loop", "0", 		   	 // loop the webp
			"-vsync", "passthrough", // Prevents frame dropping
			"-lossless", "1", 	     // lossless compression
		)
	}
	if outputExt != ".webp" {
		args = append(args, "-quality", "best")
	}

	args = append(
		args, 
		outputPath,
	)
	// convert the frames to a gif or a video
	cmd := exec.Command(ffmpegPath, args...)
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		os.Remove(outputPath)
		utils.LogError(
			err, 
			fmt.Sprintf("Pixiv: Failed to convert ugoira to %s", outputPath),
			false,
		)
		return
	}

	// delete unzipped folder which contains 
	// the frames images and the delays text file
	os.RemoveAll(imagesFolderPath)
}

// Downloads multiple Ugoira artworks and converts them based on the output format
func DownloadMultipleUgoira(downloadInfo []Ugoira, outputFormat, ffmpegPath string, deleteZip bool, ugoiraQuality int, cookies []http.Cookie) {
	outputFormat = strings.ToLower(outputFormat)
	var urlsToDownload []map[string]string
	for _, ugoira := range downloadInfo {
		urlsToDownload = append(urlsToDownload, map[string]string{
			"url": ugoira.Url,
			"filepath": ugoira.FilePath,
		})
	}

	request.DownloadURLsParallel(
		urlsToDownload, 
		utils.PIXIV_MAX_CONCURRENT_DOWNLOADS, 
		cookies, 
		GetPixivRequestHeaders(), 
		nil,
	)
	bar := utils.GetProgressBar(
		len(downloadInfo), 
		fmt.Sprintf("Converting Ugoira to %s...", outputFormat),
		utils.GetCompletionFunc(fmt.Sprintf("Finished converting %d Ugoira to %s!", len(downloadInfo), outputFormat)),
	)
	for _, downloadInfo := range downloadInfo {
		zipFilePath := filepath.Join(downloadInfo.FilePath, utils.GetLastPartOfURL(downloadInfo.Url))
		outputPath := utils.RemoveExtFromFilename(zipFilePath) + outputFormat
		if utils.PathExists(outputPath) {
			bar.Add(1)
			continue
		}

		if !utils.PathExists(zipFilePath) {
			bar.Add(1)
			continue
		}
		unzipFolderPath := filepath.Join(filepath.Dir(zipFilePath), "unzipped")
		err := utils.UnzipFile(zipFilePath, unzipFolderPath, true)
		if err != nil {
			errorMsg := fmt.Sprintf("Pixiv: Failed to unzip file %v", zipFilePath)
			utils.LogError(err, errorMsg, false)
			bar.Add(1)
			continue
		}

		ConvertUgoira(downloadInfo, unzipFolderPath, outputPath, ffmpegPath, ugoiraQuality)
		if deleteZip {
			os.Remove(zipFilePath)
		}
		bar.Add(1)
	}
}

// Query Pixiv's API for all the illustrator's posts
func GetIllustratorPosts(illustratorId, artworkType string, cookies []http.Cookie) []string {
	artworkType = strings.ToLower(artworkType)
	url := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/profile/all", illustratorId)
	res, err := request.CallRequest("GET", url, 5, cookies, GetPixivRequestHeaders(), nil, false)
	if err != nil {
		utils.LogError(err, "Pixiv: Failed to get illustrator's posts with an ID of " + illustratorId, false)
		return nil
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		errorMsg := fmt.Sprintf(
			"Pixiv: Failed to get illustrator's posts with an ID of %s due to a %s response",
			illustratorId, res.Status,
		)
		utils.LogError(nil, errorMsg, false)
		return nil
	}
	jsonBody := utils.LoadJsonFromResponse(res).(map[string]interface{})["body"]

	var artworkIds []string
	if artworkType == "all" || artworkType == "illust_and_ugoira" {
		illusts := jsonBody.(map[string]interface{})["illusts"]
		if reflect.TypeOf(illusts).Kind() == reflect.Map {
			for illustId := range illusts.(map[string]interface{}) {
				artworkIds = append(artworkIds, illustId)
			}
		}
	}

	if artworkType == "all" || artworkType == "manga" {
		manga := jsonBody.(map[string]interface{})["manga"]
		if reflect.TypeOf(manga).Kind() == reflect.Map {
			for mangaId := range manga.(map[string]interface{}) {
				artworkIds = append(artworkIds, mangaId)
			}
		}
	}
	return artworkIds
}

// Get posts from multiple illustrators and returns a map and a slice for Ugoira structures for downloads
func GetMultipleIllustratorPosts(illustratorIds []string, downloadPath, artworkType string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	bar := utils.GetProgressBar(
		len(illustratorIds), 
		"Getting artwork details from illustrator(s)...",
		utils.GetCompletionFunc(fmt.Sprintf("Finished getting artwork details from %d illustrators!", len(illustratorIds))),
	)
	var artworkIdsArr []string
	for _, illustratorId := range illustratorIds {
		artworkIds := GetIllustratorPosts(illustratorId, artworkType, cookies)
		artworkIdsArr = append(artworkIdsArr, artworkIds...)
		bar.Add(1)
	}
	artworksArr, ugoiraArr := GetMultipleArtworkDetails(artworkIdsArr, downloadPath, cookies)
	return artworksArr, ugoiraArr
}

type PixivTag struct {
	Body struct {
		IllustManga struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
		} `json:"illustManga"`
	} `json:"body"`
}

// Process the tag search results JSON and returns a slice of artwork IDs
func ProcessTagJsonResults(res *http.Response) []string {
	var pixivTagJson PixivTag
	resBody := utils.ReadResBody(res)
	err := json.Unmarshal(resBody, &pixivTagJson)
	if err != nil {
		utils.LogError(err, fmt.Sprintf("Failed to unmarshal json for Pixiv's Tag JSON\nJSON: %s", string(resBody)), false)
	}

	artworksArr := []string{}
	for _, illust := range pixivTagJson.Body.IllustManga.Data {
		artworksArr = append(artworksArr, illust.Id)
	}
	return artworksArr
}

// Query Pixiv's API and search for posts based on the supplied tag name
// which will return a map and a slice of Ugoira structures for downloads
func TagSearch(tagName, downloadPath, sortOrder, searchMode, ratingMode, artworkType string, minPage, maxPage int, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	searchMode = strings.ToLower(searchMode)
	sortOrder = strings.ToLower(sortOrder)
	ratingMode = strings.ToLower(ratingMode)
	artworkType = strings.ToLower(artworkType)

	url := "https://www.pixiv.net/ajax/search/artworks/" + tagName
	params := map[string]string{
		"word": tagName,	  // search term
		"s_mode": searchMode, // search mode: s_tag, s_tag_full, s_tc
		"order": sortOrder,   // sort order: date, popular, popular_male, popular_female 
							  // 			(add "_d" suffix for descending order, e.g. date_d)
		"mode": ratingMode,	  //  r18, safe, or all for both
		"type": artworkType,  // illust_and_ugoira, manga, all
	}

	if minPage > maxPage {
		minPage, maxPage = maxPage, minPage
	}

	var artworkIds []string
	headers := GetPixivRequestHeaders()
	for page := minPage; page <= maxPage; page++ {
		params["p"] = strconv.Itoa(page) // page number
		res, err := request.CallRequest("GET", url, 5, cookies, headers, params, true)
		if err != nil {
			utils.LogError(err, "Pixiv: Failed to get search results for " + tagName, false)
			continue
		}
		artworkIds = append(artworkIds, ProcessTagJsonResults(res)...)
		if page != maxPage {
			PixivSleep()
		}
	}

	artworkArr, ugoiraArr := GetMultipleArtworkDetails(artworkIds, downloadPath, cookies)
	return artworkArr, ugoiraArr
}