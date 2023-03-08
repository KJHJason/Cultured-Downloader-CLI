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
		cmd 		   *cobra.Command
		overwriteVar   *bool
		cookieFileVar  *string
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
					"Regardless of this flag, the program will overwrite any",
					"incomplete downloaded file by verifying against the Content-Length header response,",
					"if the size of locally downloaded file does not",
					"match with the header response, the program will re-download the file.",
					"Otherwise, if the Content-Length header is not present in the response,",
					"the program will only skip the file if the file already exists.",
					"However, if there's a need to overwrite the skipped files, you can use this flag to do so.",
					"You should only use this for websites like Pixiv Fanbox",
					"which does not return a Content-Length header in their response.",
					"However, do note that if you have an anti-virus program running,",
					"it might detect this program as a ransomware due as it is overwriting multiple files at once.",
					"Thus, it is recommended to not use this flag if you have an",
					"anti-virus program unless the anti-virus software has been disabled.",
				},
			),
		)
		cmd.Flags().StringVar(
			cmdInfo.cookieFileVar,
			"cookie_file",
			"",
			utils.CombineStringsWithNewline(
				[]string{
					"Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.",
					"You can generate a cookie file by using the \"Get cookies.txt\" extension for your browser.",
				},
			),
		)
	}
}
