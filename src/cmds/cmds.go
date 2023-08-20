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
	cmd                     *cobra.Command
	overwriteVar            *bool
	cookieFileVar           *string
	userAgentVar            *string
	gdriveApiKeyVar         *string 
	gdriveServiceAccPathVar *string
	logUrlsVar              *bool
	textFile                textFilePath
}

func init() {
	commonCmdFlags := [...]commonFlags{
		{
			cmd: fantiaCmd,
			overwriteVar:            &fantiaOverwrite,
			cookieFileVar:           &fantiaCookieFile,
			userAgentVar:            &fantiaUserAgent,
			gdriveApiKeyVar:         &fantiaGdriveApiKey,
			gdriveServiceAccPathVar: &fantiaGdriveServiceAccPath,
			logUrlsVar:              &fantiaLogUrls,
			textFile: textFilePath {
				variable: &fantiaDlTextFile,
				desc:     "Path to a text file containing Fanclub and/or post URL(s) to download from Fantia.",
			},
		},
		{
			cmd: pixivFanboxCmd,
			overwriteVar:            &fanboxOverwriteFiles,
			cookieFileVar:           &fanboxCookieFile,
			userAgentVar:            &fanboxUserAgent,
			gdriveApiKeyVar:         &fanboxGdriveApiKey,
			gdriveServiceAccPathVar: &fanboxGdriveApiKey,
			logUrlsVar:              &fanboxLogUrls,
			textFile: textFilePath {
				variable: &fanboxDlTextFile,
				desc:     "Path to a text file containing creator and/or post URL(s) to download from Pixiv Fanbox.",
			},
		},
		{
			cmd: pixivCmd,
			overwriteVar:  &pixivOverwrite,
			cookieFileVar: &pixivCookieFile,
			userAgentVar:  &pixivUserAgent,
			textFile: textFilePath {
				variable: &pixivDlTextFile,
				desc:     "Path to a text file containing artwork, illustrator, and tag name URL(s) to download from Pixiv.",
			},
		},
		{
			cmd: kemonoCmd,
			overwriteVar:            &kemonoOverwrite,
			cookieFileVar:           &kemonoCookieFile,
			userAgentVar:            &kemonoUserAgent,
			gdriveApiKeyVar:         &kemonoGdriveApiKey,
			gdriveServiceAccPathVar: &kemonoGdriveServiceAccPath,
			logUrlsVar:              &kemonoLogUrls,
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
				"Overwrite any existing files if there is no Content-Length header in the response.",
				"Usually used for Pixiv Fanbox when there are incomplete downloads.",
			),
		)
		cmd.Flags().StringVarP(
			cmdInfo.userAgentVar,
			"user_agent",
			"u",
			"",
			"Set a custom User-Agent header to use when communicating with the API(s) or when downloading.",
		)
		cmd.Flags().StringVarP(
			cmdInfo.textFile.variable,
			"txt_filepath",
			"p",
			"",
			cmdInfo.textFile.desc,
		)
		cmd.Flags().StringVarP(
			cmdInfo.cookieFileVar,
			"cookie_file",
			"c",
			"",
			utils.CombineStringsWithNewline(
				"Pass in a file path to your saved Netscape/Mozilla generated cookie file to use when downloading.",
				"You can generate a cookie file by using the \"Get cookies.txt LOCALLY\" extension for your browser.",
				"Chrome Extension URL: https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc",
			),
		)
		if cmdInfo.gdriveApiKeyVar != nil {
			cmd.Flags().StringVar(
				cmdInfo.gdriveApiKeyVar,
				"gdrive_api_key",
				"",
				utils.CombineStringsWithNewline(
					"Google Drive API key to use for downloading gdrive files.",
					"Guide: https://github.com/KJHJason/Cultured-Downloader/blob/main/doc/google_api_key_guide.md",
				),
			)
		}
		if cmdInfo.gdriveServiceAccPathVar != nil {
			cmd.Flags().StringVar(
				cmdInfo.gdriveServiceAccPathVar,
				"gdrive_service_acc_path",
				"",
				utils.CombineStringsWithNewline(
					"Path to the Google Drive service account JSON file to use for downloading gdrive files.",
					"Generally, this is preferred over the API key as it is less likely to be flagged as bot traffic.",
					"Guide: WIP",
				),
			)
		}
		if cmdInfo.logUrlsVar != nil {
			cmd.Flags().BoolVarP(
				cmdInfo.logUrlsVar,
				"log_urls",
				"l",
				false,
				utils.CombineStringsWithNewline(
					"Log any detected URLs of the files that are being downloaded.",
					"Note that not all URLs are logged, only URLs to external file hosting providers like MEGA, Google Drive, etc. are logged.",
				),
			)
		}
		RootCmd.AddCommand(cmd)
	}
}
