package cmds

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	downloadPath string
	RootCmd      = &cobra.Command{
		Use:     "cultured-downloader-cli",
		Version: fmt.Sprintf(
			"%s by KJHJason\n%s", 
			utils.VERSION, 
			"GitHub Repo: https://github.com/KJHJason/Cultured-Downloader-CLI",
		),
		Short:   "Download images, videos, etc. from various websites like Fantia.",
		Long:    "Cultured Downloader CLI is a command-line tool for downloading images, videos, etc. from various websites like Pixiv, Pixiv Fanbox, and Fantia.",
		Run: func(cmd *cobra.Command, args []string) {
			if downloadPath != "" {
				err := utils.SetDefaultDownloadPath(downloadPath)
				if err != nil {
					color.Red(err.Error())
				} else {
					color.Green("Download path set to: %s", downloadPath)
				}
			}
		},
	}
)
