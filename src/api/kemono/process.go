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

func getInlineImages(content, postFolderPath, tld string) []*request.ToDownload {
	var toDownload []*request.ToDownload
	for _, match := range imgSrcTagRegex.FindAllStringSubmatch(content, -1) {
		imgSrc := match[imgSrcTagRegexIdx]
		if imgSrc == "" {
			continue
		}
		toDownload = append(toDownload, &request.ToDownload{
			Url:      getKemonoUrl(tld) + imgSrc,
			FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER, utils.GetLastPartOfUrl(imgSrc)),
		})
	}
	return toDownload
}

// Since the name of each attachment or file is not always the filename of the file as it could be a URL,
// we need to check if the returned name value is a URL and if it is, we just return the postFolderPath as the file path.
func getKemonoFilePath(postFolderPath, childDir, fileName string) string {
	if strings.HasPrefix(fileName, "http://") || strings.HasPrefix(fileName, "https://") {
		return filepath.Join(postFolderPath, childDir)
	}
	return filepath.Join(postFolderPath, childDir, fileName)
}

func processJson(resJson *models.MainKemonoJson, tld, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	postFolderPath := utils.GetPostFolder(
		filepath.Join(downloadPath, "Kemono-Party", resJson.Service),
		resJson.User,
		resJson.Id,
		resJson.Title,
	)

	var gdriveLinks []*request.ToDownload
	var toDownload []*request.ToDownload
	if dlOptions.DlAttachments {
		toDownload = getInlineImages(resJson.Content, postFolderPath, tld)
		for _, attachment := range resJson.Attachments {
			toDownload = append(toDownload, &request.ToDownload{
				Url:      getKemonoUrl(tld) + attachment.Path,
				FilePath: getKemonoFilePath(postFolderPath, utils.KEMONO_CONTENT_FOLDER, attachment.Name),
			})
		}

		if resJson.Embed.Url != "" {
			embedsDirPath := filepath.Join(postFolderPath, utils.KEMONO_EMBEDS_FOLDER)
			if dlOptions.Configs.LogUrls {
				utils.DetectOtherExtDLLink(resJson.Embed.Url, embedsDirPath)
			}
			if utils.DetectGDriveLinks(resJson.Embed.Url, postFolderPath, true, dlOptions.Configs.LogUrls,) && dlOptions.DlGdrive {
				gdriveLinks = append(gdriveLinks, &request.ToDownload{
					Url:      resJson.Embed.Url,
					FilePath: embedsDirPath,
				})
			}
		}

		if resJson.File.Path != "" { 
			// usually is the thumbnail of the post
			toDownload = append(toDownload, &request.ToDownload{
				Url:      getKemonoUrl(tld) + resJson.File.Path,
				FilePath: getKemonoFilePath(postFolderPath, "", resJson.File.Name),
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

func processMultipleJson(resJson models.KemonoJson, tld, downloadPath string, dlOptions *KemonoDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	var urlsToDownload, gdriveLinks []*request.ToDownload
	for _, post := range resJson {
		toDownload, foundGdriveLinks := processJson(post, tld, downloadPath, dlOptions)
		urlsToDownload = append(urlsToDownload, toDownload...)
		gdriveLinks = append(gdriveLinks, foundGdriveLinks...)
	}
	return urlsToDownload, gdriveLinks
}
