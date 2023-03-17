package pixivfanbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/KJHJason/Cultured-Downloader-CLI/api/pixivfanbox/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/request"
	"github.com/KJHJason/Cultured-Downloader-CLI/spinner"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Returns a defined request header needed to communicate with Pixiv Fanbox's API
func GetPixivFanboxHeaders() map[string]string {
	return map[string]string{
		"Origin":  utils.PIXIV_FANBOX_URL,
		"Referer": utils.PIXIV_FANBOX_URL,
	}
}

// Process and detects for any external download links from the post's text content
func processPixivFanboxText(postBodyStr, postFolderPath string, downloadGdrive bool) []*request.ToDownload {
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
				postBodyStr := strings.Join(postBodySlice, "\n")
				utils.LogMessageToPath(
					"Found potential password in the post:\n\n"+postBodyStr,
					filePath,
					utils.ERROR,
				)
			}
		}

		utils.DetectOtherExtDLLink(text, postFolderPath)
		if utils.DetectGDriveLinks(text, postFolderPath, false) && downloadGdrive {
			detectedGdriveLinks = append(detectedGdriveLinks, &request.ToDownload{
				Url:      text,
				FilePath: filepath.Join(postFolderPath, utils.GDRIVE_FOLDER),
			})
		}
	}
	return detectedGdriveLinks
}

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
		detectedGdriveLinks := processPixivFanboxText(
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
		detectedGdriveLinks := processPixivFanboxText(
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
		detectedGdriveLinks := processPixivFanboxText(
			textContent.Text,
			postFolderPath,
			pixivFanboxDlOptions.DlGdrive,
		)
		if detectedGdriveLinks != nil {
			gdriveLinks = append(gdriveLinks, detectedGdriveLinks...)
		}
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

// Query Pixiv Fanbox's API based on the slice of post IDs and
// returns a map of urls and a map of GDrive urls to download from.
func (pf *PixivFanboxDl) getPostDetails(config *configs.Config, pixivFanboxDlOptions *PixivFanboxDlOptions) ([]*request.ToDownload, []*request.ToDownload) {
	maxConcurrency := utils.MAX_API_CALLS
	postIdsLen := len(pf.PostIds)
	if postIdsLen < maxConcurrency {
		maxConcurrency = postIdsLen
	}
	var wg sync.WaitGroup
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *http.Response, postIdsLen)
	errChan := make(chan error, postIdsLen)

	baseMsg := "Getting post details from Pixiv Fanbox [%d/" + fmt.Sprintf("%d]...", postIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting %d post details from Pixiv Fanbox!",
			postIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting %d post details from Pixiv Fanbox.\nPlease refer to the logs for more details.",
			postIdsLen,
		),
		postIdsLen,
	)
	progress.Start()

	url := fmt.Sprintf("%s/post.info", utils.PIXIV_FANBOX_API_URL)
	for _, postId := range pf.PostIds {
		wg.Add(1)
		queue <- struct{}{}
		go func(postId string) {
			defer func() {
				<-queue
				wg.Done()
			}()

			header := GetPixivFanboxHeaders()
			params := map[string]string{"postId": postId}
			res, err := request.CallRequest(
				&request.RequestArgs{
					Method:    "GET",
					Url:       url,
					Cookies:   pixivFanboxDlOptions.SessionCookies,
					Headers:   header,
					Params:    params,
					UserAgent: config.UserAgent,
				},
			)
			if err != nil {
				errChan <- fmt.Errorf(
					"pixiv fanbox error %d: failed to get post details for %s, more info => %v",
					utils.CONNECTION_ERROR,
					url,
					err,
				)
			} else if res.StatusCode != 200 {
				errChan <- fmt.Errorf(
					"pixiv fanbox error %d: failed to get post details for %s due to a %s response",
					utils.CONNECTION_ERROR,
					url,
					res.Status,
				)
			} else {
				resChan <- res
			}
			progress.MsgIncrement(baseMsg)
		}(postId)
	}
	close(queue)
	wg.Wait()
	close(resChan)
	close(errChan)

	hasErr := false
	if len(errChan) > 0 {
		hasErr = true
		utils.LogErrors(false, errChan, utils.ERROR)
	}
	progress.Stop(hasErr)

	// parse the responses
	var errSlice []error
	var urlsMap, gdriveUrls []*request.ToDownload
	baseMsg = "Processing received JSON(s) from Pixiv Fanbox [%d/" + fmt.Sprintf("%d]...", len(resChan))
	progress = spinner.New(
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
		postUrls, postGdriveLinks, err := processFanboxPost(
			res,
			utils.DOWNLOAD_PATH,
			pixivFanboxDlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			urlsMap = append(urlsMap, postUrls...)
			gdriveUrls = append(gdriveUrls, postGdriveLinks...)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr = false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)

	return urlsMap, gdriveUrls
}

type resStruct struct {
	json *models.FanboxCreatorPostsJson
	err  error
}

// GetFanboxCreatorPosts returns a slice of post IDs for a given creator
func getFanboxPosts(creatorId, pageNum string, config *configs.Config, dlOption *PixivFanboxDlOptions) ([]string, error) {
	params := map[string]string{"creatorId": creatorId}
	headers := GetPixivFanboxHeaders()
	url := fmt.Sprintf(
		"%s/post.paginateCreator",
		utils.PIXIV_FANBOX_API_URL,
	)
	res, err := request.CallRequest(
		&request.RequestArgs{
			Method:    "GET",
			Url:       url,
			Cookies:   dlOption.SessionCookies,
			Headers:   headers,
			Params:    params,
			UserAgent: config.UserAgent,
		},
	)
	if err != nil || res.StatusCode != 200 {
		const errPrefix = "pixiv fanbox error"
		if err != nil {
			err = fmt.Errorf(
				"%s %d: failed to get creator's posts for %s due to %v",
				errPrefix,
				utils.CONNECTION_ERROR,
				creatorId,
				err,
			)
		} else {
			res.Body.Close()
			err = fmt.Errorf(
				"%s %d: failed to get creator's posts for %s due to %s response",
				errPrefix,
				utils.RESPONSE_ERROR,
				creatorId,
				res.Status,
			)
		}
		return nil, err
	}

	var resJson models.CreatorPaginatedPostsJson
	resBody, err := utils.ReadResBody(res)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resBody, &resJson)
	if err != nil {
		err = fmt.Errorf(
			"pixiv fanbox error %d: failed to unmarshal json for Pixiv Fanbox creator's pages for %s\nJSON: %s",
			utils.JSON_ERROR,
			creatorId,
			string(resBody),
		)
		return nil, err
	}
	paginatedUrls := resJson.Body

	minPage, maxPage, hasMax, err := utils.GetMinMaxFromStr(pageNum)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	maxConcurrency := utils.MAX_API_CALLS
	if len(paginatedUrls) < maxConcurrency {
		maxConcurrency = len(paginatedUrls)
	}
	queue := make(chan struct{}, maxConcurrency)
	resChan := make(chan *resStruct, len(paginatedUrls))
	for idx, paginatedUrl := range paginatedUrls {
		curPage := idx + 1
		if curPage < minPage {
			continue
		}
		if hasMax && curPage > maxPage {
			break
		}

		wg.Add(1)
		queue <- struct{}{}
		go func(reqUrl string) {
			defer func() {
				<-queue
				wg.Done()
			}()
			res, err := request.CallRequest(
				&request.RequestArgs{
					Method:    "GET",
					Url:       reqUrl,
					Cookies:   dlOption.SessionCookies,
					Headers:   headers,
					UserAgent: config.UserAgent,
				},
			)
			if err != nil || res.StatusCode != 200 {
				if err == nil {
					res.Body.Close()
				}
				utils.LogError(
					err,
					fmt.Sprintf("failed to get post for %s", reqUrl),
					false,
					utils.ERROR,
				)
				return
			}

			var resJson *models.FanboxCreatorPostsJson
			err = utils.LoadJsonFromResponse(res, &resJson)
			if err != nil {
				resChan <- &resStruct{err: err}
				return
			}

			resChan <- &resStruct{json: resJson}
		}(paginatedUrl)
	}
	close(queue)
	wg.Wait()
	close(resChan)

	// parse the JSON response
	var errSlice []error
	var postIds []string
	for res := range resChan {
		if res.err != nil {
			errSlice = append(errSlice, res.err)
			continue
		}

		for _, postInfoMap := range res.json.Body.Items {
			postIds = append(postIds, postInfoMap.Id)
		}
	}

	if len(errSlice) > 0 {
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	return postIds, nil
}

// Retrieves all the posts based on the slice of creator IDs and updates its slice of post IDs accordingly
func (pf *PixivFanboxDl) getCreatorsPosts(config *configs.Config, dlOptions *PixivFanboxDlOptions) {
	creatorIdsLen := len(pf.CreatorIds)
	if creatorIdsLen != len(pf.CreatorPageNums) {
		panic(
			fmt.Errorf(
				"pixiv fanbox error %d: length of creator IDs and page numbers are not equal",
				utils.DEV_ERROR,
			),
		)
	}

	var errSlice []error
	baseMsg := "Getting post ID(s) from creator(s) on Pixiv Fanbox [%d/" + fmt.Sprintf("%d]...", creatorIdsLen)
	progress := spinner.New(
		spinner.REQ_SPINNER,
		"fgHiYellow",
		fmt.Sprintf(
			baseMsg,
			0,
		),
		fmt.Sprintf(
			"Finished getting post ID(s) from %d creator(s) on Pixiv Fanbox!",
			creatorIdsLen,
		),
		fmt.Sprintf(
			"Something went wrong while getting post IDs from %d creator(s) on Pixiv Fanbox!\nPlease refer to logs for more details.",
			creatorIdsLen,
		),
		creatorIdsLen,
	)
	progress.Start()
	for idx, creatorId := range pf.CreatorIds {
		retrievedPostIds, err := getFanboxPosts(
			creatorId,
			pf.CreatorPageNums[idx],
			config,
			dlOptions,
		)
		if err != nil {
			errSlice = append(errSlice, err)
		} else {
			pf.PostIds = append(pf.PostIds, retrievedPostIds...)
		}
		progress.MsgIncrement(baseMsg)
	}

	hasErr := false
	if len(errSlice) > 0 {
		hasErr = true
		utils.LogErrors(false, nil, utils.ERROR, errSlice...)
	}
	progress.Stop(hasErr)
	pf.PostIds = utils.RemoveSliceDuplicates(pf.PostIds)
}

// Start the download process for Pixiv Fanbox
func PixivFanboxDownloadProcess(config *configs.Config, pixivFanboxDl *PixivFanboxDl, pixivFanboxDlOptions *PixivFanboxDlOptions) {
	if !pixivFanboxDlOptions.DlThumbnails && !pixivFanboxDlOptions.DlImages && !pixivFanboxDlOptions.DlAttachments && !pixivFanboxDlOptions.DlGdrive {
		return
	}

	if len(pixivFanboxDl.CreatorIds) > 0 {
		pixivFanboxDl.getCreatorsPosts(
			config,
			pixivFanboxDlOptions,
		)
	}

	var urlsToDownload, gdriveUrlsToDownload []*request.ToDownload
	if len(pixivFanboxDl.PostIds) > 0 {
		urlsToDownload, gdriveUrlsToDownload = pixivFanboxDl.getPostDetails(
			config,
			pixivFanboxDlOptions,
		)
	}

	if len(urlsToDownload) > 0 {
		request.DownloadUrls(
			urlsToDownload,
			&request.DlOptions{
				MaxConcurrency: utils.PIXIV_MAX_CONCURRENT_DOWNLOADS,
				Headers:        GetPixivFanboxHeaders(),
				Cookies:        pixivFanboxDlOptions.SessionCookies,
				UseHttp3:       false,
			},
			config,
		)
	}
	if pixivFanboxDlOptions.GDriveClient != nil && len(gdriveUrlsToDownload) > 0 {
		pixivFanboxDlOptions.GDriveClient.DownloadGdriveUrls(gdriveUrlsToDownload, config)
	}
}
