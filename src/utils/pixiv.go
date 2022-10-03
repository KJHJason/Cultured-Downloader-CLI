package utils

import (
	"os"
	"fmt"
	"sort"
	"sync"
	"strconv"
	"os/exec"
	"strings"
	"net/http"
	"path/filepath"
	"github.com/panjf2000/ants/v2"
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

func ProcessArtworkJson(artworkUrlsJson interface{}, illustType int64, postDownloadDir string) ([]map[string]string, Ugoira) {
	if artworkUrlsJson == nil {
		return []map[string]string{}, Ugoira{}
	}

	var urlsToDownload []map[string]string
	var ugoiraDownloadInfo Ugoira
	artworkUrls := artworkUrlsJson.(map[string]interface{})["body"]
	if illustType == ugoira {
		ugoiraMap := artworkUrls.(map[string]interface{})
		originalUrl := ugoiraMap["originalSrc"].(string)

		// map the files to their delay
		frameInfoMap := map[string]int64{}
		for _, frame := range ugoiraMap["frames"].([]map[string]interface{}) {
			frameInfoMap[frame["file"].(string)] = int64(frame["delay"].(float64))
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

func GetArtworkDetails(artworkId, downloadPath string, cookies []http.Cookie) (string, interface{}, int64) {
	headers := GetPixivRequestHeaders()
	url := fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s", artworkId)
	artworkDetailsRes, err := CallRequest("GET", url, 15, nil, headers, nil, false)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to get artwork details for %v", artworkId)
		LogError(err, errorMsg, false)
		return "", nil, -1
	}
	if artworkDetailsRes == nil {
		return "", nil, 0
	}
	defer artworkDetailsRes.Body.Close()
	if artworkDetailsRes.StatusCode != 200 {
		errorMsg := fmt.Sprintf(
			"Failed to get details for artwork ID %v due to %s response",
			artworkId,
			artworkDetailsRes.Status,
		)
		LogError(nil, errorMsg, false)
		return "", nil, 0
	}
	artworkDetailsJson := LoadJsonFromResponse(artworkDetailsRes).(map[string]interface{})["body"]
	if artworkDetailsJson == nil {
		return "", nil, 0
	}
	artworkJsonBody := artworkDetailsJson.(map[string]interface{})
	illustratorName := artworkJsonBody["userName"].(string)
	artworkName := artworkJsonBody["title"].(string)
	artworkPostDir := CreatePostFolder(filepath.Join(downloadPath, "Pixiv"), illustratorName, artworkId, artworkName)

	artworkType := int64(artworkJsonBody["illustType"].(float64))
	switch artworkType {
	case illust, manga: // illustration or manga
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%v/pages", artworkId)
	case ugoira: // ugoira
		url = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%v/ugoira_meta", artworkId)
	default:
		return "", nil, 0
	}

	artworkUrlsRes, err := CallRequest("GET", url, 15, nil, headers, nil, false)
	if err == nil {
		return "", nil, 0
	}
	defer artworkUrlsRes.Body.Close()
	if artworkUrlsRes.StatusCode != 200 {
		errorMsg := fmt.Sprintf(
			"Failed to get details for artwork ID %v due to %s response",
			artworkId,
			artworkUrlsRes.Status,
		)
		LogError(nil, errorMsg, false)
		return "", nil, 0
	}
	return artworkPostDir, LoadJsonFromResponse(artworkUrlsRes), artworkType
}

func ConvertUgoira(ugoiraInfo Ugoira, imagesFolderPath, outputPath string, ffmpegPath string) {
	outputExtIsAllowed := false
	outputExt := filepath.Ext(outputPath)
	for _, ext := range UGOIRA_ACCEPTED_EXT {
		if outputExt == ext {
			outputExtIsAllowed = true
			break
		}
	}
	if !outputExtIsAllowed {
		panic(fmt.Sprintf("Output extension %v is not allowed for ugoira conversion", outputExt))
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
	// copy the last frame and add it to the end of the imagePath map
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

	// ffmpeg flags: https://www.ffmpeg.org/ffmpeg.html
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
			args = append(args, "-lossless", "1") 
		}

		args = append(
			args,
			"-c:v", encodingMap[outputExt], // video codec
			"-vsync", "vfr", 				// variable frame rate
		)
	} else {
		args = append(
			args,
			"-loop", "0", // loop the gif
		)
	}

	if outputExt != ".webm" {
		crf := "10" // https://superuser.com/questions/677576/what-is-crf-used-for-in-ffmpeg
					// the lower the crf value the better the quality
		args = append(args, "-crf", crf)
	}

	args = append(
		args, 
		"-quality", "best",
		outputPath,
	)
	// convert the frames to a gif or a video
	cmd := exec.Command(ffmpegPath, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		os.Remove(outputPath)
		LogError(
			err, 
			fmt.Sprintf("Failed to convert ugoira to %s", outputPath),
			false,
		)
		return
	}

	// delete unzipped folder which contains 
	// the frames images and the delays text file
	os.Remove(concatDelayFilePath)
	os.RemoveAll(imagesFolderPath)
}

func DownloadUgoira(downloadInfo []Ugoira, deleteZip bool, cookies []http.Cookie) {
	var urlsToDownload []map[string]string
	for _, ugoira := range downloadInfo {
		urlsToDownload = append(urlsToDownload, map[string]string{
			"url": ugoira.Url,
			"filepath": ugoira.FilePath,
		})
	}

	DownloadURLsParallel(urlsToDownload, cookies, GetPixivRequestHeaders(), nil)
	for _, downloadInfo := range downloadInfo {
		zipFilePath := filepath.Join(downloadInfo.FilePath, GetLastPartOfURL(downloadInfo.Url))
		gifPath := strings.TrimSuffix(zipFilePath, filepath.Ext(zipFilePath)) + ".gif"
		if PathExists(gifPath) {
			continue
		}

		if !PathExists(zipFilePath) {
			continue
		}
		unzipFolderPath := filepath.Join(filepath.Dir(zipFilePath), "unzipped")
		err := UnzipFile(zipFilePath, unzipFolderPath, true)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to unzip file %v", zipFilePath)
			LogError(err, errorMsg, false)
			continue
		}

		ConvertUgoira(downloadInfo, unzipFolderPath, gifPath, "ffmpeg")
		if deleteZip {
			os.Remove(zipFilePath)
		}
	}
}

func GetIllustratorPosts(illustratorId, artworkType string, cookies []http.Cookie) []string {
	url := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/profile/all", illustratorId)
	res, err := CallRequest("GET", url, 5, cookies, GetPixivRequestHeaders(), nil, false)
	if err != nil {
		LogError(err, "Failed to get illustrator's posts with an ID of " + illustratorId, false)
		return nil
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		errorMsg := fmt.Sprintf(
			"Failed to get illustrator's posts with an ID of %s due to a %s response",
			illustratorId, res.Status,
		)
		LogError(nil, errorMsg, false)
		return nil
	}
	jsonBody := LoadJsonFromResponse(res).(map[string]interface{})["body"]

	var artworkIds []string
	if artworkType == "all" || artworkType == "illust_and_ugoira" {
		illusts := jsonBody.(map[string]interface{})["illusts"]
		if illusts != nil {
			for illustId := range illusts.(map[string]interface{}) {
				artworkIds = append(artworkIds, illustId)
			}
		}
	}

	if artworkType == "all" || artworkType == "manga" {
		manga := jsonBody.(map[string]interface{})["manga"]
		if manga != nil {
			for mangaId := range manga.(map[string]interface{}) {
				artworkIds = append(artworkIds, mangaId)
			}
		}
	}
	return artworkIds
}

func ProcessTagJsonResults(jsonInterfaces []interface{}) ([]string, []string) {
	ugoiraMap := []string{}
	illustAndMangaMap := []string{}
	for _, jsonInterface := range jsonInterfaces {
		illustsBody := jsonInterface.(map[string]interface{})["body"]
		illustsData := illustsBody.(map[string]interface{})["illustManga"].(map[string]interface{})["data"]
		if illustsData == nil {
			continue
		}

		for _, illust := range illustsData.([]interface{}) {
			illustMap := illust.(map[string]interface{})
			illustType := int64(illustMap["illustType"].(float64))
			if illustType == ugoira {
				ugoiraMap = append(ugoiraMap, illustMap["id"].(string))
			} else {
				illustAndMangaMap = append(illustAndMangaMap, illustMap["id"].(string))
			}
		}
	}
	return ugoiraMap, illustAndMangaMap
}

func TagSearch(tagName, sortOrder, searchMode, mode, artworkType string, minPage, maxPage int, cookies []http.Cookie) []interface{} {
	url := "https://www.pixiv.net/ajax/search/artworks/" + tagName
	params := map[string]string{
		"word": tagName,	  // search term
		"s_mode": searchMode, // search mode: s_tag, s_tag_full, s_tc
		"order": sortOrder,   // sort order: date, popular, popular_male, popular_female 
							  // 			(add "_d" suffix for descending order, e.g. date_d)
		"mode": mode,		  //  r18, safe, or all for both
		"type": artworkType,  // illust_and_ugoira, manga, all
	}

	if minPage > maxPage {
		minPage, maxPage = maxPage, minPage
	}

	var wg sync.WaitGroup
	maxConcurrency := MAX_API_CALLS
	pageDiff := maxPage - minPage + 1
	if pageDiff < MAX_API_CALLS {
		maxConcurrency = pageDiff
	}

	resChan := make(chan *http.Response, pageDiff)
	pool, _ := ants.NewPool(maxConcurrency)
	defer pool.Release()
	for page := minPage; page <= maxPage; page++ {
		wg.Add(1)
		params["p"] = strconv.Itoa(page) // page number
		err := pool.Submit(func() {
			defer wg.Done()
			resp, err := CallRequest("GET", url, 5, cookies, GetPixivRequestHeaders(), params, true)
			if err != nil {
				LogError(err, "Pixiv: Failed to get search results for " + tagName, false)
				return
			}
			resChan <- resp
		})
		if err != nil {
			panic(err)
		}
	}
	wg.Wait()
	close(resChan)

	var artworks []interface{}
	for res := range resChan {
		artworks = append(artworks, LoadJsonFromResponse(res))
	}
	return artworks
}