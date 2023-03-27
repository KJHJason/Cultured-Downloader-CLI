package gdrive

import (
	"path/filepath"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Process and detects for any external download links from the post's text content
func ProcessPostText(postBodyStr, postFolderPath string, downloadGdrive bool, logUrls bool) []*request.ToDownload {
	if postBodyStr == "" {
		return nil
	}

	// split the text by newlines
	postBodySlice := strings.FieldsFunc(
		postBodyStr,
		func(c rune) bool {
			return c == '\n'
		},
	)
	loggedPassword := false
	var detectedGdriveLinks []*request.ToDownload
	for _, text := range postBodySlice {
		if utils.DetectPasswordInText(text) && !loggedPassword {
			// Log the entire post text if it contains a password
			filePath := filepath.Join(postFolderPath, utils.PASSWORD_FILENAME)
			if !utils.PathExists(filePath) {
				loggedPassword = true
				utils.LogMessageToPath(
					"Found potential password in the post:\n\n" + postBodyStr,
					filePath,
					utils.ERROR,
				)
			}
		}

		if logUrls {
			utils.DetectOtherExtDLLink(text, postFolderPath)
		}	
		if utils.DetectGDriveLinks(text, postFolderPath, false, logUrls) && downloadGdrive {
			detectedGdriveLinks = append(detectedGdriveLinks, &request.ToDownload{
				Url:      text,
				FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
			})
		}
	}
	return detectedGdriveLinks
}
