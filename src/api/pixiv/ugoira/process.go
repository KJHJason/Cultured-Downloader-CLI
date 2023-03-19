package ugoira

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/common"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Map the Ugoira frame delays to their respective filenames
func MapDelaysToFilename(ugoiraFramesJson models.UgoiraFramesJson) map[string]int64 {
	frameInfoMap := map[string]int64{}
	for _, frame := range ugoiraFramesJson {
		frameInfoMap[frame.File] = int64(frame.Delay)
	}
	return frameInfoMap
}

type UgoiraFfmpegArgs struct {
	ffmpegPath    string
	outputPath    string
	ugoiraQuality int
}

// Converts the Ugoira to the desired output path using FFmpeg
func ConvertUgoira(ugoiraInfo *models.Ugoira, imagesFolderPath string, ugoiraFfmpeg *UgoiraFfmpegArgs) error {
	outputExt := filepath.Ext(ugoiraFfmpeg.outputPath)
	if !utils.SliceContains(UGOIRA_ACCEPTED_EXT, outputExt) {
		return fmt.Errorf(
			"pixiv error %d: Output extension %v is not allowed for ugoira conversion",
			utils.INPUT_ERROR,
			outputExt,
		)
	}

	concatDelayFilePath, sortedFilenames, err := writeDelays(ugoiraInfo, imagesFolderPath)
	if err != nil {
		return err
	}

	args, err := getFfmpegFlagsForUgoira(
		&ffmpegOptions{
			ffmpegPath:          ugoiraFfmpeg.ffmpegPath,
			outputExt:           outputExt,
			concatDelayFilePath: concatDelayFilePath,
			sortedFilenames:     sortedFilenames,
			outputPath:          ugoiraFfmpeg.outputPath,
			ugoiraQuality:       ugoiraFfmpeg.ugoiraQuality,
		},
		imagesFolderPath,
	)
	if err != nil {
		return err
	}

	// convert the frames to a gif or a video
	cmd := exec.Command(ugoiraFfmpeg.ffmpegPath, args...)
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		os.Remove(ugoiraFfmpeg.outputPath)
		err = fmt.Errorf(
			"pixiv error %d: failed to convert ugoira to %s, more info => %v",
			utils.CMD_ERROR,
			ugoiraFfmpeg.outputPath,
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
	filePath := filepath.Join(ugoireFilePath, utils.GetLastPartOfUrl(ugoiraUrl))
	outputFilePath := utils.RemoveExtFromFilename(filePath) + outputFormat
	return filePath, outputFilePath
}

func convertMultipleUgoira(ugoiraArgs *UgoiraArgs, ugoiraOptions *UgoiraOptions, config *configs.Config) {
	// Create a context that can be cancelled when SIGINT/SIGTERM signal is received
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Catch SIGINT/SIGTERM signal and cancel the context when received
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()
	defer signal.Stop(sigs)

	var errSlice []error
	downloadInfoLen := len(ugoiraArgs.ToDownload)
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
	for i, ugoira := range ugoiraArgs.ToDownload {
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
		err := utils.ExtractFiles(ctx, zipFilePath, unzipFolderPath, true)
		if err != nil {
			if err == context.Canceled {
				progress.KillProgram(
					fmt.Sprintf(
						"Stopped converting ugoira to %s [%d/%d]!", 
						ugoiraOptions.OutputFormat, 
						i, 
						len(ugoiraArgs.ToDownload),
					),
				)
			}
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
			&UgoiraFfmpegArgs{
				ffmpegPath: config.FfmpegPath,
				outputPath: outputPath,
				ugoiraQuality: ugoiraOptions.Quality,
			},
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
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)
}

type UgoiraArgs struct {
	UseMobileApi  bool
	ToDownload    []*models.Ugoira
	Cookies       []*http.Cookie
}

// Downloads multiple Ugoira artworks and converts them based on the output format
func DownloadMultipleUgoira(ugoiraArgs *UgoiraArgs, ugoiraOptions *UgoiraOptions, config *configs.Config, reqHandler request.RequestHandler) {
	var urlsToDownload []*request.ToDownload
	for _, ugoira := range ugoiraArgs.ToDownload {
		filePath, outputFilePath := GetUgoiraFilePaths(
			ugoira.FilePath,
			ugoira.Url,
			ugoiraOptions.OutputFormat,
		)
		if !utils.PathExists(outputFilePath) {
			urlsToDownload = append(urlsToDownload, &request.ToDownload{
				Url:      ugoira.Url,
				FilePath: filePath,
			})
		}
	}

	var useHttp3 bool
	var headers map[string]string
	if ugoiraArgs.UseMobileApi {
		headers = map[string]string{
			"Referer": "https://app-api.pixiv.net",
		}
	} else {
		headers = pixivcommon.GetPixivRequestHeaders()
		useHttp3 = utils.IsHttp3Supported(utils.PIXIV, true)
	}

	request.DownloadUrlsWithHandler(
		urlsToDownload,
		&request.DlOptions{
			MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
			Headers:        headers,
			Cookies:        ugoiraArgs.Cookies,
			UseHttp3:       useHttp3,
		},
		config,    // Note: if isMobileApi is true, custom user-agent will be ignored
		reqHandler,
	)

	convertMultipleUgoira(ugoiraArgs, ugoiraOptions, config)
}
