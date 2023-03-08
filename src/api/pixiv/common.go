package pixiv

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type Ugoira struct {
	Url      string
	FilePath string
	Frames   map[string]int64
}

// Returns a defined request header needed to communicate with Pixiv's API
func GetPixivRequestHeaders() map[string]string {
	return map[string]string{
		"Origin":  "https://www.pixiv.net/",
		"Referer": "https://www.pixiv.net/",
	}
}

func GetIllustUrl(illustId string) string {
	return "https://www.pixiv.net/artworks/" + illustId
}
func GetUserUrl(userId string) string {
// Map the Ugoira frame delays to their respective filenames
func MapDelaysToFilename(ugoiraFramesJson map[string]interface{}) map[string]int64 {
	frameInfoMap := map[string]int64{}
	for _, frame := range ugoiraFramesJson["frames"].([]interface{}) {
		frameMap := frame.(map[string]interface{})
		frameInfoMap[frameMap["file"].(string)] = int64(frameMap["delay"].(float64))
	}
	return frameInfoMap
}

// Converts the Ugoira to the desired output path using FFmpeg
func ConvertUgoira(ugoiraInfo Ugoira, imagesFolderPath, outputPath string, ffmpegPath string, ugoiraQuality int) error {
	outputExt := filepath.Ext(outputPath)
	if !utils.SliceContains(UGOIRA_ACCEPTED_EXT, outputExt) {
		return fmt.Errorf("pixiv error %d: Output extension %v is not allowed for ugoira conversion", utils.INPUT_ERROR, outputExt)
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
		return fmt.Errorf(
			"pixiv error %d: failed to create delays.txt, more info => %v",
			utils.OS_ERROR,
			err,
		)
	}

	_, err = f.WriteString(delaysText)
	if err != nil {
		f.Close()
		return fmt.Errorf(
			"pixiv error %d: failed to write delay string to delays.txt, more info => %v",
			utils.OS_ERROR,
			err,
		)
	}
	f.Close()

	// FFmpeg flags: https://www.ffmpeg.org/ffmpeg.html
	args := []string{
		"-y",           // overwrite output file if it exists
		"-an",          // disable audio
		"-f", "concat", // input is a concat file
		"-safe", "0", // allow absolute paths in the concat file
		"-i", concatDelayFilePath, // input file
	}
	if outputExt == ".webm" || outputExt == ".mp4" {
		// if converting the ugoira to a webm or .mp4 file
		// then set the output video codec to vp9 or h264 respectively
		// 	- webm: https://trac.ffmpeg.org/wiki/Encode/VP9
		// 	- mp4: https://trac.ffmpeg.org/wiki/Encode/H.264
		encodingMap := map[string]string{
			".webm": "libvpx-vp9",
			".mp4":  "libx264",
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
			"-pix_fmt", "yuv420p", // set the pixel format to yuv420p
			"-c:v", encodingMap[outputExt], // video codec
			"-vsync", "passthrough", // Prevents frame dropping
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
			err = fmt.Errorf(
				"pixiv error %d: failed to generate palette for ugoira gif, more info => %v",
				utils.CMD_ERROR,
				err,
			)
			return err
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
			"-plays", "0", // loop the apng
			"-vf",
			"setpts=PTS-STARTPTS,hqdn3d=1.5:1.5:6:6", // set the setpts filter and apply some denoising
		)
	} else { // outputExt == ".webp"
		args = append(
			args,
			"-pix_fmt", "yuv420p", // set the pixel format to yuv420p
			"-loop", "0", // loop the webp
			"-vsync", "passthrough", // Prevents frame dropping
			"-lossless", "1", // lossless compression
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
		err = fmt.Errorf(
			"pixiv error %d: failed to convert ugoira to %s, more info => %v",
			utils.CMD_ERROR,
			outputPath,
			err,
		)
		return err
	}

	// delete unzipped folder which contains
	// the frames images and the delays text file
	os.RemoveAll(imagesFolderPath)
	return nil
}

// Returns the ugoira's zip file path and the ugoira's converted file path
func GetUgoiraFilePaths(ugoireFilePath, ugoiraUrl, outputFormat string) (string, string) {
	filePath := filepath.Join(ugoireFilePath, utils.GetLastPartOfURL(ugoiraUrl))
	outputFilePath := utils.RemoveExtFromFilename(filePath) + outputFormat
	return filePath, outputFilePath
}

// Downloads multiple Ugoira artworks and converts them based on the output format
func downloadMultipleUgoira(
	downloadInfo *[]Ugoira, config *api.Config, ugoiraOptions *UgoiraOptions, cookies []http.Cookie,
) {
	var urlsToDownload []map[string]string
	for _, ugoira := range *downloadInfo {
		filePath, outputFilePath := GetUgoiraFilePaths(
			ugoira.FilePath,
			ugoira.Url,
			ugoiraOptions.OutputFormat,
		)
		if !utils.PathExists(outputFilePath) {
			urlsToDownload = append(urlsToDownload, map[string]string{
				"url":      ugoira.Url,
				"filepath": filePath,
			})
		}
	}

	request.DownloadURLsParallel(
		&urlsToDownload,
		utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
		cookies,
		GetPixivRequestHeaders(),
		nil,
		config.OverwriteFiles,
	)

	var errSlice []error
	downloadInfoLen := len(*downloadInfo)
	baseMsg := "Converting Ugoira to %s [%d/" + fmt.Sprintf("%d]...", downloadInfoLen)
	progress := spinner.New(
		spinner.DL_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished converting %d Ugoira to %s!",
			downloadInfoLen,
			ugoiraOptions.OutputFormat,
		),
		fmt.Sprintf(
			"Something went wrong while converting %d Ugoira to %s!\nPlease refer to the logs for more details.",
			downloadInfoLen,
			ugoiraOptions.OutputFormat,
		),
		downloadInfoLen,
	)
	progress.Start()
	for _, ugoira := range *downloadInfo {
		zipFilePath, outputPath := GetUgoiraFilePaths(ugoira.FilePath, ugoira.Url, ugoiraOptions.OutputFormat)
		if utils.PathExists(outputPath) {
			progress.MsgIncrement(baseMsg)
			continue
		}
		if !utils.PathExists(zipFilePath) {
			progress.MsgIncrement(baseMsg)
			continue
		}

		unzipFolderPath := filepath.Join(
			filepath.Dir(zipFilePath),
			"unzipped",
		)
		err := utils.ExtractFiles(zipFilePath, unzipFolderPath, true)
		if err != nil {
			err := fmt.Errorf(
				"pixiv error %d: failed to unzip file %s, more info => %v",
				utils.OS_ERROR,
				zipFilePath,
				err,
			)
			errSlice = append(errSlice, err)
			progress.MsgIncrement(baseMsg)
			continue
		}

		err = ConvertUgoira(
			ugoira,
			unzipFolderPath,
			outputPath,
			config.FfmpegPath,
			ugoiraOptions.Quality,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else if ugoiraOptions.DeleteZip {
			os.Remove(zipFilePath)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, errSlice...)
	}
	progress.Stop(hasErr)
}
