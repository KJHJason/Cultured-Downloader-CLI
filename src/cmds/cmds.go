package cmds

import (
	"github.com/spf13/cobra"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

func getMultipleIdsMsg() string {
	return "For multiple IDs, separate them with a comma.\nExample: \"12345,67891\" (without the quotes)"
}

type textFilePath struct {
	variable *string
	desc     string
}
type commonFlags struct {
	cmd           *cobra.Command
	overwriteVar  *bool
	cookieFileVar *string
	userAgentVar  *string
	textFile      textFilePath
}

func init() {
	commonCmdFlags := []commonFlags{
		{
			cmd: fantiaCmd,
			overwriteVar:  &fantiaOverwrite,
			cookieFileVar: &fantiaCookieFile,
			userAgentVar:  &fantiaUserAgent,
			textFile: textFilePath {
				variable: &fantiaDlTextFile,
				desc: "Path to a text file containing Fanclub and/or post URL(s) to download from Fantia.",
			},
		},
		{
			cmd: pixivFanboxCmd,
			overwriteVar:  &fanboxOverwriteFiles,
			cookieFileVar: &fanboxCookieFile,
			userAgentVar:  &fanboxUserAgent,
			textFile: textFilePath {
				variable: &fanboxDlTextFile,
				desc: "Path to a text file containing creator and/or post URL(s) to download from Pixiv Fanbox.",
			},
		},
		{
			cmd: pixivCmd,
			overwriteVar:  &pixivOverwrite,
			cookieFileVar: &pixivCookieFile,
			userAgentVar:  &pixivUserAgent,
			textFile: textFilePath {
				variable: &pixivDlTextFile,
				desc: "Path to a text file containing artwork, illustrator, and tag name URL(s) to download from Pixiv.",
			},
		},
		{
			cmd: kemonoCmd,
			overwriteVar:  &kemonoOverwrite,
			cookieFileVar: &kemonoCookieFile,
			userAgentVar:  &kemonoUserAgent,
			textFile: textFilePath {
				variable: &kemonoDlTextFile,
				desc: "Path to a text file containing creator and/or post URL(s) to download from Kemono Party.",
			},
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
		cmd.Flags().StringVar(
			cmdInfo.userAgentVar,
			"user_agent",
			"",
			"Set a custom User-Agent header to use when communicating with the API(s) or when downloading.",
		)
		cmd.Flags().StringVar(
			cmdInfo.textFile.variable,
			"text_file",
			"",
			cmdInfo.textFile.desc,
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
		RootCmd.AddCommand(cmd)
	}
}
