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

func init() {
	RootCmd.Flags().StringVar(
		&downloadPath,
		"download_path",
		"",
		utils.CombineStringsWithNewline(
			[]string{
				"Configure the path to download the files to and save it for future runs.",
				"Otherwise, the program will use the current working directory.",
				"Note:",
				"If you had used the \"-download_path\" flag before or",
				"had used the Cultured Downloader Python program, the program will automatically use the path you had set.",
			},
		),
	)
	RootCmd.CompletionOptions.HiddenDefaultCmd = true
	RootCmd.AddCommand(
		fantiaCmd,
		pixivFanboxCmd,
		pixivCmd,
	)
}
