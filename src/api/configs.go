package api

import (
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
)

type Config struct {
	GDriveClient   *gdrive.GDrive
	DownloadPath   string
	FfmpegPath     string
	OverwriteFiles bool
	UserAgent      string
}

func (c *Config) ValidateFfmpeg() {
	_, ffmpegErr := exec.LookPath(c.FfmpegPath)
	if ffmpegErr != nil {
		color.Red("FFmpeg is not installed.\nPlease install it from https://ffmpeg.org/ and either use the -ffmpeg_path flag or add the FFmpeg path to your PATH environment variable or alias depending on your OS.")
		os.Exit(1)
	}
}
