package kemono

import (
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

func processJson(resJson *models.MainKemonoJson, downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	postFolderPath := utils.GetPostFolder(
		filepath.Join(downloadPath, "Kemono-Party", resJson.Service),
		resJson.User,
		resJson.Id,
		resJson.Title,
	)

	var toDownload []*request.ToDownload
	if dlOption.DlAttachments {
		toDownload = getInlineImages(resJson.Content, postFolderPath)
		for _, attachment := range resJson.Attachments {
			toDownload = append(toDownload, &request.ToDownload{
				Url:      utils.KEMONO_URL + attachment.Path,
				FilePath: filepath.Join(postFolderPath, attachment.Name),
			})
		}
		toDownload = append(toDownload, &request.ToDownload{
			Url:      utils.KEMONO_URL + resJson.File.Path,
			FilePath: filepath.Join(postFolderPath, resJson.File.Name),
		})
	}

	gdriveLinks := gdrive.ProcessPostText(
		resJson.Content,
		postFolderPath,
		dlOption.DlGdrive,
	)
	return toDownload, gdriveLinks
}

func processMultipleJson(resJson models.KemonoJson, downloadPath string, dlOption *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var urlsToDownload, gdriveLinks []*request.ToDownload
	for _, post := range resJson {
		toDownload, foundGdriveLinks := processJson(post, downloadPath, dlOption)
		urlsToDownload = append(urlsToDownload, toDownload...)
		gdriveLinks = append(gdriveLinks, foundGdriveLinks...)
	}
	return urlsToDownload, gdriveLinks
}
