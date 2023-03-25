package ugoira

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixiv/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type ffmpegOptions struct {
	ffmpegPath          string
	outputExt           string
	concatDelayFilePath string
	sortedFilenames     []string
	ugoiraQuality       int
	outputPath          string
}

func writeDelays(ugoiraInfo *models.Ugoira, imagesFolderPath string) (string, []string, error) {
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
		return "", nil, fmt.Errorf(
			"pixiv error %d: failed to create delays.txt, more info => %v",
			utils.OS_ERROR,
			err,
		)
	}
	defer f.Close()

	_, err = f.WriteString(delaysText)
	if err != nil {
		return "", nil, fmt.Errorf(
			"pixiv error %d: failed to write delay string to delays.txt, more info => %v",
			utils.OS_ERROR,
			err,
		)
	}
	return concatDelayFilePath, sortedFilenames, nil
}

func getFlagsForWebmAndMp4(outputExt string, ugoiraQuality int) []string {
	var args []string
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

	// if converting the ugoira to a webm or .mp4 file
	// then set the output video codec to vp9 or h264 respectively
	// 	- webm: https://trac.ffmpeg.org/wiki/Encode/VP9
	// 	- mp4: https://trac.ffmpeg.org/wiki/Encode/H.264
	var encoding string
	if outputExt == ".webm" {
		encoding = "libvpx-vp9"
	} else { // outputExt == ".mp4"
		encoding = "libx264"
	}

	args = append(
		args,
		"-pix_fmt", "yuv420p",     // set the pixel format to yuv420p
		"-c:v",     encoding,      // video codec
		"-vsync",   "passthrough", // Prevents frame dropping
	)
	return args
}

func getFlagsForGif(options *ffmpegOptions, imagesFolderPath string) ([]string, error) {
	// Generate a palette for the gif using FFmpeg for better quality
	palettePath := filepath.Join(imagesFolderPath, "palette.png")
	ffmpegImages := "%" + fmt.Sprintf(
		"%dd%s", // Usually it's %6d.extension but just in case, measure the length of the filename
		len(utils.RemoveExtFromFilename(options.sortedFilenames[0])),
		filepath.Ext(options.sortedFilenames[0]),
	)
	imagePaletteCmd := exec.Command(
		options.ffmpegPath,
		"-i", filepath.Join(imagesFolderPath, ffmpegImages),
		"-vf", "palettegen",
		palettePath,
	)

	if utils.DEBUG_MODE {
		imagePaletteCmd.Stdout = os.Stdout
		imagePaletteCmd.Stderr = os.Stderr
	}
	err := imagePaletteCmd.Run()
	if err != nil {
		return nil, fmt.Errorf(
			"pixiv error %d: failed to generate palette for ugoira gif, more info => %v",
			utils.CMD_ERROR,
			err,
		)
	}
	return []string{
		"-loop", "0", // loop the gif
		"-i",    palettePath,
		"-filter_complex", "paletteuse",
	}, nil
}

func getFfmpegFlagsForUgoira(options *ffmpegOptions, imagesFolderPath string) ([]string, error) {
	// FFmpeg flags: https://www.ffmpeg.org/ffmpeg.html
	args := []string{
		"-y",                                 // overwrite output file if it exists
		"-an",                                // disable audio
		"-f",    "concat",                    // input is a concat file
		"-safe", "0",                         // allow absolute paths in the concat file
		"-i",    options.concatDelayFilePath, // input file
	}
	switch options.outputExt {
	case ".webm", ".mp4":
		args = append(args, getFlagsForWebmAndMp4(options.outputExt, options.ugoiraQuality)...)
	case ".gif":
		gifArgs, err := getFlagsForGif(options, imagesFolderPath)
		if err != nil {
			return nil, err
		}
		args = append(args, gifArgs...)
	case ".apng":
		args = append(
			args,
			"-plays", "0", // loop the apng
			"-vf",
			"setpts=PTS-STARTPTS,hqdn3d=1.5:1.5:6:6", // set the setpts filter and apply some denoising
		)
	case ".webp": // outputExt == ".webp"
		args = append(
			args,
			"-pix_fmt", "yuv420p", // set the pixel format to yuv420p
			"-loop", "0", // loop the webp
			"-vsync", "passthrough", // Prevents frame dropping
			"-lossless", "1", // lossless compression
		)
	default:
		panic(
			fmt.Sprintf(
				"pixiv error %d: Output extension %v is not allowed for ugoira conversion",
				utils.DEV_ERROR,
				options.outputExt,
			),
		)
	}
	if options.outputExt != ".webp" {
		args = append(args, "-quality", "best")
	}

	args = append(args, options.outputPath)
	return args, nil
}
