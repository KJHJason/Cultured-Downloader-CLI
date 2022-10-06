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
	"sync"
	"reflect"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
)

const (
	illust = iota
	manga
	ugoira
)

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

type ArtworkDetailsJsonRes struct {
	ArtworkFolderPath string
	JsonRes interface{}
	ArtworkType int64
}

func GetMultipleArtworkDetails(artworkIds []string, downloadPath string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	maxConcurrency := utils.PIXIV_MAX_CONCURRENT_API_CALLS
	if len(artworkIds) < maxConcurrency {
		maxConcurrency = len(artworkIds)
	}
	var wg sync.WaitGroup
	resChan := make(chan ArtworkDetailsJsonRes, len(artworkIds))
	queue := make(chan struct{}, maxConcurrency)
	for _, artworkId := range artworkIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(artworkId string) {
			defer wg.Done()
			artworkFolder, jsonRes, artworkType := GetArtworkDetails(artworkId, downloadPath, cookies)
			if artworkFolder != "" && artworkType != -1 {
				resChan <- ArtworkDetailsJsonRes{
					ArtworkFolderPath: artworkFolder,
					JsonRes: jsonRes,
					ArtworkType: artworkType,
				}
			}
			<-queue
		}(artworkId)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	var ugoiraDetails []Ugoira
	var artworkDetails []map[string]string
	for res := range resChan {
		processedJsonRes, ugoiraInfo := ProcessArtworkJson(res.JsonRes, res.ArtworkType, res.ArtworkFolderPath)
		if res.ArtworkType == ugoira {
			ugoiraDetails = append(ugoiraDetails, ugoiraInfo)
		} else {
			artworkDetails = append(artworkDetails, processedJsonRes...)
		}
	}
	return artworkDetails, ugoiraDetails
}

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

	concatDelayFilePath := filepath.Join(filepath.Dir(outputPath), "delays.txt")
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
		"-an",					 	// disable audio
		"-f", "concat", 			// input is a concat file
		"-safe", "0",   			// allow absolute paths in the concat file
		"-i", concatDelayFilePath,	// input file
	}
	if outputExt != ".gif" {
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
			args = append(
				args,
				"-pix_fmt", "yuv420p", 					// pixel format
				"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2", // pad the video to be even
			)
		} else {
			// webm will be loseless but takes longer to convert
			if ugoiraQuality == 0 {
				args = append(args, "-lossless", "1") 
			} else {
				args = append(args, "-crf", strconv.Itoa(ugoiraQuality))
			}
		}

		args = append(
			args,
			"-c:v", encodingMap[outputExt], // video codec
			"-vsync", "vfr", 				// variable frame rate
		)
	} else {
		// Generate a palette for the gif using FFmpeg for better quality
		palettePath := filepath.Join(imagesFolderPath, "palette.png")
		imagePaletteCmd := exec.Command(
			ffmpegPath,
			"-i", filepath.Join(imagesFolderPath, sortedFilenames[0]),
			"-vf", "palettegen",
			palettePath,
		)
		err = imagePaletteCmd.Run()
		if err != nil {
			panic(err)
		}
		args = append(
			args,
			"-loop", "0", // loop the gif
			"-i", palettePath,
			"-filter_complex", "paletteuse",
		)
	}

	if outputExt != ".webm" {
		args = append(args, "-crf", strconv.Itoa(ugoiraQuality))
	}

	args = append(
		args, 
		"-quality", "best",
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
	os.Remove(concatDelayFilePath)
	os.RemoveAll(imagesFolderPath)
}

func DownloadUgoira(downloadInfo []Ugoira, outputFormat, ffmpegPath string, deleteZip bool, ugoiraQuality int, cookies []http.Cookie) {
	outputFormat = strings.ToLower(outputFormat)
	var urlsToDownload []map[string]string
	for _, ugoira := range downloadInfo {
		urlsToDownload = append(urlsToDownload, map[string]string{
			"url": ugoira.Url,
			"filepath": ugoira.FilePath,
		})
	}

	request.DownloadURLsParallel(urlsToDownload, utils.PIXIV_MAX_CONCURRENT_DOWNLOADS, cookies, GetPixivRequestHeaders(), nil)
	for _, downloadInfo := range downloadInfo {
		zipFilePath := filepath.Join(downloadInfo.FilePath, utils.GetLastPartOfURL(downloadInfo.Url))
		outputPath := strings.TrimSuffix(zipFilePath, filepath.Ext(zipFilePath)) + outputFormat
		if utils.PathExists(outputPath) {
			continue
		}

		if !utils.PathExists(zipFilePath) {
			continue
		}
		unzipFolderPath := filepath.Join(filepath.Dir(zipFilePath), "unzipped")
		err := utils.UnzipFile(zipFilePath, unzipFolderPath, true)
		if err != nil {
			errorMsg := fmt.Sprintf("Pixiv: Failed to unzip file %v", zipFilePath)
			utils.LogError(err, errorMsg, false)
			continue
		}

		ConvertUgoira(downloadInfo, unzipFolderPath, outputPath, ffmpegPath, ugoiraQuality)
		if deleteZip {
			os.Remove(zipFilePath)
		}
	}
}

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

func GetMultipleIllustratorPosts(illustratorIds []string, downloadPath, artworkType string, cookies []http.Cookie) ([]map[string]string, []Ugoira) {
	maxConcurrency := utils.PIXIV_MAX_CONCURRENT_API_CALLS
	if len(illustratorIds) < maxConcurrency {
		maxConcurrency = len(illustratorIds)
	}
	var wg sync.WaitGroup
	var mut sync.Mutex
	var artworkIdsArr []string
	queue := make(chan struct{}, maxConcurrency)
	for _, illustratorId := range illustratorIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(illustratorId string) {
			defer wg.Done()
			artworkIds := GetIllustratorPosts(illustratorId, artworkType, cookies)
			mut.Lock()
			artworkIdsArr = append(artworkIdsArr, artworkIds...)
			mut.Unlock()
			<-queue
		}(illustratorId)
	}
	close(queue)
	wg.Wait()

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

	maxConcurrency := utils.PIXIV_MAX_CONCURRENT_API_CALLS
	pageDiff := maxPage - minPage + 1
	if pageDiff < maxConcurrency {
		maxConcurrency = pageDiff
	}

	var wg sync.WaitGroup
	resChan := make(chan *http.Response, pageDiff)
	queue := make(chan struct{}, maxConcurrency)
	headers := GetPixivRequestHeaders()
	for page := minPage; page <= maxPage; page++ {
		wg.Add(1)
		params["p"] = strconv.Itoa(page) // page number
		queue <- struct{}{}
		go func(params map[string]string) {
			defer wg.Done()
			resp, err := request.CallRequest("GET", url, 5, cookies, headers, params, true)
			if err != nil {
				utils.LogError(err, "Pixiv: Failed to get search results for " + tagName, false)
				return
			}
			resChan <- resp
			<-queue
		}(params)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	var artworkIds []string
	for res := range resChan {
		artworkIds = append(artworkIds, ProcessTagJsonResults(res)...)
	}
	artworkArr, ugoiraArr := GetMultipleArtworkDetails(artworkIds, downloadPath, cookies)
	return artworkArr, ugoiraArr
}