package pixivfanbox

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
)

// Pixiv Fanbox permitted file extensions based on
// https://fanbox.pixiv.help/hc/en-us/articles/360011057793-What-types-of-attachments-can-I-post-
var pixivFanboxAllowedImageExt = []string{"jpg", "jpeg", "png", "gif"}

// Process the JSON response from Pixiv Fanbox's API and
// returns a map of urls and a map of GDrive urls to download from
func processFanboxPost(res *http.Response, downloadPath string, pixivFanboxDlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	var err error
	var post models.FanboxPostJson
	var postJsonBody []byte
	err = utils.LoadJsonFromResponse(res, &post)
	if err != nil {
		return nil, nil, err
	}

	postJson := post.Body
	postId := postJson.Id
	postTitle := postJson.Title
	creatorId := postJson.CreatorId
	postFolderPath := utils.GetPostFolder(
		filepath.Join(downloadPath, "Pixiv-Fanbox"),
		creatorId,
		postId,
		postTitle,
	)

	var urlsMap []*request.ToDownload
	thumbnail := postJson.CoverImageUrl
	if pixivFanboxDlOptions.DlThumbnails && thumbnail != "" {
		urlsMap = append(urlsMap, &request.ToDownload{
			Url:      thumbnail,
			FilePath: postFolderPath,
		})
	}

	// Note that Pixiv Fanbox posts have 3 types of formatting (as of now):
	//	1. With proper formatting and mapping of post content elements ("article")
	//	2. With a simple formatting that obly contains info about the text and files ("file", "image")
	postType := postJson.Type
	postBody := postJson.Body
	if postBody == nil {
		return urlsMap, nil, nil
	}

	var gdriveLinks []*request.ToDownload
	switch postType {
	case "file":
		// process the text in the post
		filePostJson := postBody.(*models.FanboxFilePostJson)
		detectedGdriveLinks := gdrive.ProcessPostText(
			filePostJson.Text,
			postFolderPath,
			pixivFanboxDlOptions.DlGdrive,
		)
		if detectedGdriveLinks != nil {
			gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
		}

		imageAndAttachmentUrls := filePostJson.Files
		if (len(imageAndAttachmentUrls) > 0) && (pixivFanboxDlOptions.DlImages || pixivFanboxDlOptions.DlAttachments) {
			for _, fileInfo := range imageAndAttachmentUrls {
				fileUrl := fileInfo.Url
				extension := fileInfo.Extension
				filename := fileInfo.Name + "." + extension

				var filePath string
				isImage := utils.SliceContains(pixivFanboxAllowedImageExt, extension)
				if isImage {
					filePath = filepath.Join(postFolderPath, utils.IMAGES_FOLDER, filename)
				} else {
					filePath = filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename)
				}

				if (isImage && pixivFanboxDlOptions.DlImages) || (!isImage && pixivFanboxDlOptions.DlAttachments) {
					urlsMap = append(urlsMap, &request.ToDownload{
						Url:      fileUrl,
						FilePath: filePath,
					})
				}
			}
		}
	case "image":
		// process the text in the post
		imagePostJson := postBody.(*models.FanboxImagePostJson)
		detectedGdriveLinks := gdrive.ProcessPostText(
			imagePostJson.Text,
			postFolderPath,
			pixivFanboxDlOptions.DlGdrive,
		)
		if detectedGdriveLinks != nil {
			gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
		}

		// retrieve images and attachments url(s)
		imageAndAttachmentUrls := imagePostJson.Images
		if (imageAndAttachmentUrls != nil) && (pixivFanboxDlOptions.DlImages || pixivFanboxDlOptions.DlAttachments) {
			for _, fileInfo := range imageAndAttachmentUrls {
				fileUrl := fileInfo.OriginalUrl
				extension := fileInfo.Extension
				filename := utils.GetLastPartOfUrl(fileUrl)

				var filePath string
				isImage := utils.SliceContains(pixivFanboxAllowedImageExt, extension)
				if isImage {
					filePath = filepath.Join(postFolderPath, utils.IMAGES_FOLDER, filename)
				} else {
					filePath = filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename)
				}

				if (isImage && pixivFanboxDlOptions.DlImages) || (!isImage && pixivFanboxDlOptions.DlAttachments) {
					urlsMap = append(urlsMap, &request.ToDownload{
						Url:      fileUrl,
						FilePath: filePath,
					})
				}
			}
		}
	case "article":
		// process the text in the post
		articleJson := postBody.(*models.FanboxArticleJson)
		articleBlocks := articleJson.Blocks
		if len(articleBlocks) > 0 {
			loggedPassword := false
			for _, articleBlock := range articleBlocks {
				text := articleBlock.Text
				if text != "" {
					if !loggedPassword && utils.DetectPasswordInText(text) {
						// Log the entire post text if it contains a password
						filePath := filepath.Join(postFolderPath, utils.PASSWORD_FILENAME)
						if !utils.PathExists(filePath) {
							loggedPassword = true
							postBodyStr := "Found potential password in the post:\n\n"
							for _, articleContent := range articleBlocks {
								articleText := articleContent.Text
								if articleText != "" {
									postBodyStr += articleText + "\n"
								}
							}
							utils.LogMessageToPath(
								postBodyStr,
								filePath,
								utils.ERROR,
							)
						}
					}

					utils.DetectOtherExtDLLink(text, postFolderPath)
					if utils.DetectGDriveLinks(text, postFolderPath, false) && pixivFanboxDlOptions.DlGdrive {
						gdriveLinks = append(gdriveLinks, &request.ToDownload{
							Url:      text,
							FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
						})
					}
				}

				articleLinks := articleBlock.Links
				if len(articleLinks) > 0 {
					for _, articleLink := range articleLinks {
						linkUrl := articleLink.Url
						utils.DetectOtherExtDLLink(linkUrl, postFolderPath)
						if utils.DetectGDriveLinks(linkUrl, postFolderPath, true) && pixivFanboxDlOptions.DlGdrive {
							gdriveLinks = append(gdriveLinks, &request.ToDownload{
								Url:      linkUrl,
								FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
							})
							continue
						}
					}
				}
			}
		}

		// retrieve images and attachments url(s)
		imageMap := articleJson.ImageMap
		if imageMap != nil && pixivFanboxDlOptions.DlImages {
			for _, imageInfo := range imageMap {
				urlsMap = append(urlsMap, &request.ToDownload{
					Url:      imageInfo.OriginalUrl,
					FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
				})
			}
		}

		attachmentMap := articleJson.FileMap
		if attachmentMap != nil && pixivFanboxDlOptions.DlAttachments {
			for _, attachmentInfo := range attachmentMap {
				attachmentUrl := attachmentInfo.Url
				filename := attachmentInfo.Name + "." + attachmentInfo.Extension
				urlsMap = append(urlsMap, &request.ToDownload{
					Url:      attachmentUrl,
					FilePath: filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename),
				})
			}
		}
	case "text": // text post
		// Usually has no content but try to detect for any external download links
		textContent := postBody.(*models.FanboxTextPostJson)
		gdriveLinks = gdrive.ProcessPostText(
			textContent.Text,
			postFolderPath,
			pixivFanboxDlOptions.DlGdrive,
		)
	default: // unknown post type
		return nil, nil, fmt.Errorf(
			"pixiv fanbox error %d: unknown post type, \"%s\"\nPixiv Fanbox post content:\n%s",
			utils.JSON_ERROR,
			postType,
			string(postJsonBody),
		)
	}
	return urlsMap, gdriveLinks, nil
}
