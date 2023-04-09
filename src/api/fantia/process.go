package fantia

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/fantia/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
)

func dlImagesFromPost(content *models.FantiaContent, postFolderPath string) []*request.ToDownload {
	var urlsSlice []*request.ToDownload

	// download images that are uploaded to their own section
	postContentPhotos := content.PostContentPhotos
	for _, image := range postContentPhotos {
		imageUrl := image.URL.Original
		urlsSlice = append(urlsSlice, &request.ToDownload{
			Url:      imageUrl,
			FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
		})
	}

	// for images that are embedded in the post content
	comment := content.Comment
	matchedStr := utils.FANTIA_IMAGE_URL_REGEX.FindAllStringSubmatch(comment, -1)
	for _, matched := range matchedStr {
		imageUrl := utils.FANTIA_URL + matched[utils.FANTIA_REGEX_URL_INDEX]
		urlsSlice = append(urlsSlice, &request.ToDownload{
			Url:      imageUrl,
			FilePath: filepath.Join(postFolderPath, utils.IMAGES_FOLDER),
		})
	}
	return urlsSlice
}

func dlAttachmentsFromPost(content *models.FantiaContent, postFolderPath string) []*request.ToDownload {
	var urlsSlice []*request.ToDownload

	// get the attachment url string if it exists
	attachmentUrl := content.AttachmentURI
	if attachmentUrl != "" {
		attachmentUrlStr := utils.FANTIA_URL + attachmentUrl
		urlsSlice = append(urlsSlice, &request.ToDownload{
			Url:      attachmentUrlStr,
			FilePath: filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER),
		})
	} else if content.DownloadUri != "" {
		// if the attachment url string does not exist,
		// then get the download url for the file
		downloadUrl := utils.FANTIA_URL + content.DownloadUri
		filename := content.Filename
		urlsSlice = append(urlsSlice, &request.ToDownload{
			Url:      downloadUrl,
			FilePath: filepath.Join(postFolderPath, utils.ATTACHMENT_FOLDER, filename),
		})
	}
	return urlsSlice
}

var errRecaptcha = fmt.Errorf("recaptcha detected for the current session")

// Process the JSON response from Fantia's API and
// returns a slice of urls and a slice of gdrive urls to download from
func processFantiaPost(res *http.Response, downloadPath string, dlOptions *FantiaDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	// processes a fantia post
	// returns a map containing the post id and the url to download the file from
	var postJson models.FantiaPost
	if err := utils.LoadJsonFromResponse(res, &postJson); err != nil {
		return nil, nil, err
	}

	if postJson.Redirect != "" {
		if postJson.Redirect != "/recaptcha" {
			return nil, nil, fmt.Errorf(
				"fantia error %d: unknown redirect url, %q", 
				utils.UNEXPECTED_ERROR, 
				postJson.Redirect,
			)
		}
		return nil, nil, errRecaptcha
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

	var urlsSlice []*request.ToDownload
	thumbnail := post.Thumb.Original
	if dlOptions.DlThumbnails && thumbnail != "" {
		urlsSlice = append(urlsSlice, &request.ToDownload{
			Url:      thumbnail,
			FilePath: postFolderPath,
		})
	}

	gdriveLinks := gdrive.ProcessPostText(
		post.Comment,
		postFolderPath,
		dlOptions.DlGdrive,
		dlOptions.Configs.LogUrls,
	)

	postContent := post.PostContents
	if postContent == nil {
		return urlsSlice, gdriveLinks, nil
	}
	for _, content := range postContent {
		commentGdriveLinks := gdrive.ProcessPostText(
			content.Comment,
			postFolderPath,
			dlOptions.DlGdrive,
			dlOptions.Configs.LogUrls,
		)
		if len(commentGdriveLinks) > 0 {
			gdriveLinks = append(gdriveLinks, commentGdriveLinks...)
		}
		if dlOptions.DlImages {
			urlsSlice = append(urlsSlice, dlImagesFromPost(&content, postFolderPath)...)
		}
		if dlOptions.DlAttachments {
			urlsSlice = append(urlsSlice, dlAttachmentsFromPost(&content, postFolderPath)...)
		}
	}
	return urlsSlice, gdriveLinks, nil
}

type processIllustArgs struct {
	res          *http.Response
	postId       string
	postIdsLen   int
	msgSuffix    string
}

// Process the JSON response to get the urls to download
func processIllustDetailApiRes(illustArgs *processIllustArgs, dlOptions *FantiaDlOptions) ([]*request.ToDownload, []*request.ToDownload, error) {
	progress := spinner.New(
		spinner.JSON_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			"Processing retrieved JSON for post %s from Fantia %s...",
			illustArgs.postId,
			illustArgs.msgSuffix,
		),
		fmt.Sprintf(
			"Finished processing retrieved JSON for post %s from Fantia %s!",
			illustArgs.postId,
			illustArgs.msgSuffix,
		),
		fmt.Sprintf(
			"Something went wrong while processing retrieved JSON for post %s from Fantia %s.\nPlease refer to the logs for more details.",
			illustArgs.postId,
			illustArgs.msgSuffix,
		),
		illustArgs.postIdsLen,
	)
	progress.Start()
	urlsToDownload, gdriveLinks, err := processFantiaPost(
		illustArgs.res,
		utils.DOWNLOAD_PATH,
		dlOptions,
	)
	if err != nil {
		progress.Stop(true)
		return nil, nil, err
	}
	progress.Stop(false)
	return urlsToDownload, gdriveLinks, nil
}
