package configs

import (
	"os"
	"os/exec"

	"github.com/fatih/color"
)

type Config struct {
	// DownloadPath will be used as the base path for all downloads
	DownloadPath   string

	// FfmpegPath is the path to the FFmpeg binary
	FfmpegPath     string

	// OverwriteFiles is a flag to overwrite existing files
	// If false, the download process will be skipped if the file already exists
	OverwriteFiles bool

	// UserAgent is the user agent to be used in the download process
	UserAgent      string
}

func (c *Config) ValidateFfmpeg() {
	_, ffmpegErr := exec.LookPath(c.FfmpegPath)
	if ffmpegErr != nil {
		color.Red("FFmpeg is not installed.\nPlease install it from https://ffmpeg.org/ and either use the --ffmpeg_path flag or add the FFmpeg path to your PATH environment variable or alias depending on your OS.")
		os.Exit(1)
	}
}
