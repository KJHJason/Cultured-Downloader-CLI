package pixivfanbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Pixiv Fanbox permitted file extensions based on
// https://fanbox.pixiv.help/hc/en-us/articles/360011057793-What-types-of-attachments-can-I-post-
var pixivFanboxAllowedImageExt = []string{"jpg", "jpeg", "png", "gif"}

func detectUrlsAndPasswordsInPost(text, postFolderPath string, articleBlocks models.FanboxArticleBlocks, dlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, bool) {
	loggedPassword := false 
	if utils.DetectPasswordInText(text) {
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

	var gdriveLinks []*request.ToDownload
	if dlOptions.Configs.LogUrls {
		utils.DetectOtherExtDLLink(text, postFolderPath)
	}
	if utils.DetectGDriveLinks(text, postFolderPath, false, dlOptions.Configs.LogUrls) && dlOptions.DlGdrive {
		gdriveLinks = append(gdriveLinks, &request.ToDownload{
			Url:      text,
			FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
		})
	}
	return gdriveLinks, loggedPassword
}

func processFanboxArticlePost(postBody json.RawMessage, postFolderPath string, dlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	var articleJson models.FanboxArticleJson
	if err := utils.LoadJsonFromBytes(postBody, &articleJson); err != nil {
		return nil, nil, err
	}

	var urlsSlice []*request.ToDownload
	var gdriveLinks []*request.ToDownload
	// retrieve images and attachments url(s)
	imageMap := articleJson.ImageMap
	if imageMap != nil && dlOptions.DlImages {
		for _, imageInfo := range imageMap {
			urlsSlice = append(urlsSlice, &request.ToDownload{
				Url:      imageInfo.OriginalUrl,
				FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
			})
		}
	}

	attachmentMap := articleJson.FileMap
	if attachmentMap != nil && dlOptions.DlAttachments {
		for _, attachmentInfo := range attachmentMap {
			attachmentUrl := attachmentInfo.Url
			filename := attachmentInfo.Name + "." + attachmentInfo.Extension
			urlsSlice = append(urlsSlice, &request.ToDownload{
				Url:      attachmentUrl,
				FilePath: filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename),
			})
		}
	}

	articleBlocks := articleJson.Blocks
	if len(articleBlocks) == 0 {
		return urlsSlice, gdriveLinks, nil
	}

	loggedPassword := false
	for _, articleBlock := range articleBlocks {
		text := articleBlock.Text
		if text != "" && !loggedPassword {
			var detectedGdriveUrls []*request.ToDownload
			detectedGdriveUrls, loggedPassword = detectUrlsAndPasswordsInPost(
				text, 
				postFolderPath, 
				articleBlocks, 
				dlOptions,
			)
			gdriveLinks = append(gdriveLinks, detectedGdriveUrls...)
		}

		articleLinks := articleBlock.Links
		if len(articleLinks) > 0 {
			for _, articleLink := range articleLinks {
				linkUrl := articleLink.Url
				utils.DetectOtherExtDLLink(linkUrl, postFolderPath)
				if utils.DetectGDriveLinks(linkUrl, postFolderPath, true, dlOptions.Configs.LogUrls) && dlOptions.DlGdrive {
					gdriveLinks = append(gdriveLinks, &request.ToDownload{
						Url:      linkUrl,
						FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
					})
					continue
				}
			}
		}
	}

	return urlsSlice, gdriveLinks, nil
}

func processFanboxFilePost(postBody json.RawMessage, postFolderPath string, dlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	var filePostJson models.FanboxFilePostJson
	if err :=  utils.LoadJsonFromBytes(postBody, &filePostJson); err != nil {
		return nil, nil, err
	}

	// process the text in the post
	var urlsSlice, gdriveLinks []*request.ToDownload
	detectedGdriveLinks := gdrive.ProcessPostText(
		filePostJson.Text,
		postFolderPath,
		dlOptions.DlGdrive,
		dlOptions.Configs.LogUrls,
	)
	if detectedGdriveLinks != nil {
		gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
	}

	imageAndAttachmentUrls := filePostJson.Files
	if !dlOptions.DlImages && !dlOptions.DlAttachments {
		return nil, nil, nil
	}

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

		if (isImage && dlOptions.DlImages) || (!isImage && dlOptions.DlAttachments) {
			urlsSlice = append(urlsSlice, &request.ToDownload{
				Url:      fileUrl,
				FilePath: filePath,
			})
		}
	}
	return urlsSlice, gdriveLinks, nil
}

func processFanboxImagePost(postBody json.RawMessage, postFolderPath string, dlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	var imagePostJson models.FanboxImagePostJson
	if err := utils.LoadJsonFromBytes(postBody, &imagePostJson); err != nil {
		return nil, nil, err
	}

	// process the text in the post
	var urlsSlice, gdriveLinks []*request.ToDownload
	detectedGdriveLinks := gdrive.ProcessPostText(
		imagePostJson.Text,
		postFolderPath,
		dlOptions.DlGdrive,
		dlOptions.Configs.LogUrls,
	)
	if detectedGdriveLinks != nil {
		gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
	}

	// retrieve images and attachments url(s)
	imageAndAttachmentUrls := imagePostJson.Images
	if !dlOptions.DlImages && !dlOptions.DlAttachments {
		return nil, nil, nil
	}

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

		if (isImage && dlOptions.DlImages) || (!isImage && dlOptions.DlAttachments) {
			urlsSlice = append(urlsSlice, &request.ToDownload{
				Url:      fileUrl,
				FilePath: filePath,
			})
		}
	}
	return urlsSlice, gdriveLinks, nil
}

// Process the JSON response from Pixiv Fanbox's API and
// returns a map of urls and a map of GDrive urls to download from
func processFanboxPostJson(res *http.Response, downloadPath string, dlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	var post models.FanboxPostJson
	if err := utils.LoadJsonFromResponse(res, &post); err != nil {
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

	var urlsSlice []*request.ToDownload
	thumbnail := postJson.CoverImageUrl
	if dlOptions.DlThumbnails && thumbnail != "" {
		urlsSlice = append(urlsSlice, &request.ToDownload{
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
		return urlsSlice, nil, nil
	}

	var err error
	var newUrlsSlice []*request.ToDownload
	var gdriveLinks []*request.ToDownload
	switch postType {
	case "file":
		newUrlsSlice, gdriveLinks, err = processFanboxFilePost(postBody, postFolderPath, dlOptions)
	case "image":
		newUrlsSlice, gdriveLinks, err = processFanboxImagePost(postBody, postFolderPath, dlOptions)
	case "article":
		newUrlsSlice, gdriveLinks, err = processFanboxArticlePost(postBody, postFolderPath, dlOptions)
	case "text": // text post
		// Usually has no content but try to detect for any external download links
		var textContent models.FanboxTextPostJson
		if err = utils.LoadJsonFromBytes(postBody, &textContent); err == nil {
			gdriveLinks = gdrive.ProcessPostText(
				textContent.Text,
				postFolderPath,
				dlOptions.DlGdrive,
				dlOptions.Configs.LogUrls,
			)
		}
	default: // unknown post type
		jsonBytes, _ := json.MarshalIndent(post, "", "\t")
		return nil, nil, fmt.Errorf(
			"pixiv fanbox error %d: unknown post type, %q\nPixiv Fanbox post content:\n%s",
			utils.JSON_ERROR,
			postType,
			string(jsonBytes),
		)
	}

	if err != nil {
		return nil, nil, err
	}
	urlsSlice = append(urlsSlice, newUrlsSlice...)
	return urlsSlice, gdriveLinks, nil
}

func processMultiplePostJson(resChan chan *http.Response, dlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	// parse the responses
	var errSlice []error
	var urlsSlice, gdriveUrls []*request.ToDownload
	baseMsg := "Processing received JSON(s) from Pixiv Fanbox [%d/" + fmt.Sprintf("%d]...", len(resChan))
	progress := spinner.New(
		spinner.JSON_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished processing %d JSON(s) from Pixiv Fanbox!",
			len(resChan),
		),
		fmt.Sprintf(
			"Something went wrong while processing %d JSON(s) from Pixiv Fanbox.\nPlease refer to the logs for more details.",
			len(resChan),
		),
		len(resChan),
	)
	progress.Start()
	for res := range resChan {
		postUrls, postGdriveLinks, err := processFanboxPostJson(
			res,
			utils.DOWNLOAD_PATH,
			dlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			urlsSlice = append(urlsSlice, postUrls...)
			gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)
	return urlsSlice, gdriveUrls
}
