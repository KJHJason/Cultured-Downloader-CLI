package fantia

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Process the JSON response from Fantia's API and
// returns a slice of urls and a slice of gdrive urls to download from
func processFantiaPost(res *http.Response, downloadPath string, fantiaDlOptions *FantiaDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	var postJson models.FantiaPost
	err := utils.LoadJsonFromResponse(res, &postJson)
	if err != nil {
		return nil, nil, err
	}

	post := postJson.Post
	postId := strconv.Itoa(post.ID)
	postTitle := post.Title
	creatorName := post.Fanclub.User.Name
	postFolderPath := utils.GetPostFolder(
		filepath.Join(
			downloadPath,
			utils.FANTIA_TITLE,
		),
		creatorName,
		postId,
		postTitle,
	)

	var urlsMap []*request.ToDownload
	thumbnail := post.Thumb.Original
	if fantiaDlOptions.DlThumbnails && thumbnail != "" {
		urlsMap = append(urlsMap, &request.ToDownload{
			Url:      thumbnail,
			FilePath: postFolderPath,
		})
	}

	gdriveLinks := gdrive.ProcessPostText(
		post.Comment,
		postFolderPath,
		fantiaDlOptions.DlGdrive,
	)

	postContent := post.PostContents
	if postContent == nil {
		return urlsMap, gdriveLinks, nil
	}
	for _, content := range postContent {
		commentGdriveLinks := gdrive.ProcessPostText(
			content.Comment,
			postFolderPath,
			fantiaDlOptions.DlGdrive,
		)
		if len(commentGdriveLinks) > 0 {
			gdriveLinks = append(gdriveLinks, commentGdriveLinks...)
		}

		if fantiaDlOptions.DlImages {
			// download images that are uploaded to their own section
			postContentPhotos := content.PostContentPhotos
			for _, image := range postContentPhotos {
				imageUrl := image.URL.Original
				urlsMap = append(urlsMap, &request.ToDownload{
					Url:      imageUrl,
					FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
				})
			}

			// for images that are embedded in the post content
			comment := content.Comment
			matchedStr := utils.FANTIA_IMAGE_URL_REGEX.FindAllStringSubmatch(comment, -1)
			for _, matched := range matchedStr {
				imageUrl := utils.FANTIA_URL + matched[utils.FANTIA_REGEX_URL_INDEX]
				urlsMap = append(urlsMap, &request.ToDownload{
					Url:      imageUrl,
					FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
				})
			}
		}

		if fantiaDlOptions.DlAttachments {
			// get the attachment url string if it exists
			attachmentUrl := content.AttachmentURI
			if attachmentUrl != "" {
				attachmentUrlStr := utils.FANTIA_URL + attachmentUrl
				urlsMap = append(urlsMap, &request.ToDownload{
					Url:      attachmentUrlStr,
					FilePath: filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER),
				})
			} else if content.DownloadUri != "" {
				// if the attachment url string does not exist,
				// then get the download url for the file
				downloadUrl := utils.FANTIA_URL + content.DownloadUri
				filename := content.Filename
				urlsMap = append(urlsMap, &request.ToDownload{
					Url:      downloadUrl,
					FilePath: filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename),
				})
			}
		}
	}
	return urlsMap, gdriveLinks, nil
}
