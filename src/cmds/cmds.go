package cmds

import (
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func getMultipleIdsMsg() string {
	return "For multiple IDs, separate them with a comma.\nExample: \"12345,67891\" (without the quotes)"
}

func init() {
	type commonFlags struct {
		cmd           *cobra.Command
		overwriteVar  *bool
		cookieFileVar *string
	}
	commonCmdFlags := []commonFlags{
		{
			cmd: fantiaCmd,
			overwriteVar: &fantiaOverwrite,
			cookieFileVar: &fantiaCookieFile,
		},
		{
			cmd: pixivFanboxCmd,
			overwriteVar: &fanboxOverwriteFiles,
			cookieFileVar: &fanboxCookieFile,
		},
		{
			cmd: pixivCmd,
			overwriteVar: &pixivOverwrite,
			cookieFileVar: &pixivCookieFile,
		},
	}
	for _, cmdInfo := range commonCmdFlags {
		cmd := cmdInfo.cmd

		cmd.Flags().BoolVarP(
			cmdInfo.overwriteVar,
			"overwrite",
			"o",
			false,
			utils.CombineStringsWithNewline(
				[]string{
					"Overwrite any existing files if there is no Content-Length header in the response.",
					"Usually used for Pixiv Fanbox when there are incomplete downloads.",
				},
			),
		)
		cmd.Flags().StringVarP(
			cmdInfo.cookieFileVar,
			"cookie_file",
			"c",
			"",
			utils.CombineStringsWithNewline(
				[]string{
					"Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.",
					"You can generate a cookie file by using the \"Get cookies.txt LOCALLY\" extension for your browser.",
					"Chrome Extension URL: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc",
				},
			),
		)
	}
}
