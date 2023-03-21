package kemono

import (
	"strings"
	"regexp"
	"path/filepath"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
)

var (
	imgSrcTagRegex = regexp.MustCompile(`(?i)<img[^>]+src=(?:\\)?"(?P<imgSrc>[^">]+)(?:\\)?"[^>]*>`)
	imgSrcTagRegexIdx = imgSrcTagRegex.SubexpIndex("imgSrc")
)

func getInlineImages(content, postFolderPath string) []*request.ToDownload {
	var toDownload []*request.ToDownload
	for _, match := range imgSrcTagRegex.FindAllStringSubmatch(content, -1) {
		imgSrc := match[imgSrcTagRegexIdx]
		if imgSrc == "" {
			continue
		}
		toDownload = append(toDownload, &request.ToDownload{
			Url:      utils.KEMONO_URL + imgSrc,
			FilePath: filepath.Join(postFolderPath, utils.GetLastPartOfUrl(imgSrc)),
		})
	}
	return toDownload
}

// Since the name of each attachment or file is not always the filename of the file as it could be a URL,
// we need to check if the returned name value is a URL and if it is, we just return the postFolderPath as the file path.
func getKemonoFilePath(postFolderPath, fileName string) string {
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		return postFolderPath
	}
	return filepath.Join(postFolderPath, fileName)
}

func processJson(resJson *models.MainKemonoJson, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	postFolderPath := utils.GetPostFolder(
		filepath.Join(downloadPath, "Kemono-Party", resJson.Service),
		resJson.User,
		resJson.Id,
		resJson.Title,
	)

	var gdriveLinks []*request.ToDownload
	var toDownload []*request.ToDownload
	if dlOptions.DlAttachments {
		toDownload = getInlineImages(resJson.Content, postFolderPath)
		for _, attachment := range resJson.Attachments {
			toDownload = append(toDownload, &request.ToDownload{
				Url:      utils.KEMONO_URL + attachment.Path,
				FilePath: getKemonoFilePath(postFolderPath, attachment.Name),
			})
		}

		if resJson.Embed.Url != "" {
			if dlOptions.Configs.LogUrls {
				utils.DetectOtherExtDLLink(resJson.Embed.Url, postFolderPath)
			}
			if utils.DetectGDriveLinks(resJson.Embed.Url, postFolderPath, true, dlOptions.Configs.LogUrls,) && dlOptions.DlGdrive {
				gdriveLinks = append(gdriveLinks, &request.ToDownload{
					Url:      resJson.Embed.Url,
					FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
				})
			}
		}

		if resJson.File.Path != "" {
			toDownload = append(toDownload, &request.ToDownload{
				Url:      utils.KEMONO_URL + resJson.File.Path,
				FilePath: getKemonoFilePath(postFolderPath, resJson.File.Name),
			})
		}
	}

	contentGdriveLinks := gdrive.ProcessPostText(
		resJson.Content,
		postFolderPath,
		dlOptions.DlGdrive,
		dlOptions.Configs.LogUrls,
	)
	gdriveLinks = append(gdriveLinks, contentGdriveLinks...)
	return toDownload, gdriveLinks
}

func processMultipleJson(resJson models.KemonoJson, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var urlsToDownload, gdriveLinks []*request.ToDownload
	for _, post := range resJson {
		toDownload, foundGdriveLinks := processJson(post, downloadPath, dlOptions)
		urlsToDownload = append(urlsToDownload, toDownload...)
		gdriveLinks = append(gdriveLinks, foundGdriveLinks...)
	}
	return urlsToDownload, gdriveLinks
}
