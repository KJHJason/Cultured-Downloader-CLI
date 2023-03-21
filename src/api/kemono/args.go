package kemono

import (
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono/models"
	"github.com/KJHJason/Cultured-Downloader-CLI/configs"
	"github.com/KJHJason/Cultured-Downloader-CLI/gdrive"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

const (
	BASE_REGEX_STR             = `https://kemono\.party/(?P<service>patreon|fanbox|gumroad|subscribestar|dlsite|fantia|boosty)/user/(?P<creatorId>[\w-]+)`
	BASE_POST_SUFFIX_REGEX_STR = `/post/(?P<postId>\d+)`
	SERVICE_GROUP_NAME         = "service"
	CREATOR_ID_GROUP_NAME      = "creatorId"
	POST_ID_GROUP_NAME         = "postId"
	API_MAX_CONCURRENT         = 3
)

var (
	POST_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%s%s$`,
			BASE_REGEX_STR,
			BASE_POST_SUFFIX_REGEX_STR,
		),
	)
	POST_URL_REGEX_SERVICE_INDEX = POST_URL_REGEX.SubexpIndex(SERVICE_GROUP_NAME)
	POST_URL_REGEX_CREATOR_ID_INDEX = POST_URL_REGEX.SubexpIndex(CREATOR_ID_GROUP_NAME)
	POST_URL_REGEX_POST_ID_INDEX = POST_URL_REGEX.SubexpIndex(POST_ID_GROUP_NAME)

	CREATOR_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%s$`,
			BASE_REGEX_STR,
		),
	)
	CREATOR_URL_REGEX_SERVICE_INDEX = CREATOR_URL_REGEX.SubexpIndex(SERVICE_GROUP_NAME)
	CREATOR_URL_REGEX_CREATOR_ID_INDEX = CREATOR_URL_REGEX.SubexpIndex(CREATOR_ID_GROUP_NAME)
)

type KemonoDl struct {
	CreatorUrls     []string
	CreatorPageNums []string
	CreatorsToDl    []*models.KemonoCreatorToDl

	PostUrls  []string
	PostsToDl []*models.KemonoPostToDl
}

func ProcessCreatorUrls(creatorUrls []string, pageNums []string) []*models.KemonoCreatorToDl {
	creatorsToDl := make([]*models.KemonoCreatorToDl, len(creatorUrls))
	for i, creatorUrl := range creatorUrls {
		matched := CREATOR_URL_REGEX.FindStringSubmatch(creatorUrl)
		creatorsToDl[i] = &models.KemonoCreatorToDl{
			Service:   matched[CREATOR_URL_REGEX_SERVICE_INDEX],
			CreatorId: matched[CREATOR_URL_REGEX_CREATOR_ID_INDEX],
			PageNum:   pageNums[i],
		}
	}

	return creatorsToDl
}

func ProcessPostUrls(postUrls []string) []*models.KemonoPostToDl {
	postsToDl := make([]*models.KemonoPostToDl, len(postUrls))
	for i, postUrl := range postUrls {
		matched := POST_URL_REGEX.FindStringSubmatch(postUrl)
		postsToDl[i] = &models.KemonoPostToDl{
			Service:   matched[POST_URL_REGEX_SERVICE_INDEX],
			CreatorId: matched[POST_URL_REGEX_CREATOR_ID_INDEX],
			PostId:    matched[POST_URL_REGEX_POST_ID_INDEX],
		}
	}

	return postsToDl
}

// RemoveDuplicates removes duplicate creators and posts from the slice
func (k *KemonoDl) RemoveDuplicates() {
	if len(k.CreatorsToDl) > 0 {
		newCreatorSlice := make([]*models.KemonoCreatorToDl, 0, len(k.CreatorsToDl))
		seen := make(map[string]struct{})
		for _, creator := range k.CreatorsToDl {
			key := fmt.Sprintf("%s/%s", creator.Service, creator.CreatorId)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			newCreatorSlice = append(newCreatorSlice, creator)
		}
		k.CreatorsToDl = newCreatorSlice
	}

	if len(k.PostsToDl) == 0 {
		return
	}
	newPostSlice := make([]*models.KemonoPostToDl, 0, len(k.PostsToDl))
	seen := make(map[string]struct{})
	for _, post := range k.PostsToDl {
		key := fmt.Sprintf("%s/%s/%s", post.Service, post.CreatorId, post.PostId)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		newPostSlice = append(newPostSlice, post)
	}
	k.PostsToDl = newPostSlice
}

func (k *KemonoDl) ValidateArgs() {
	valid, outlier := utils.SliceMatchesRegex(CREATOR_URL_REGEX, k.CreatorUrls)
	if !valid {
		color.Red(
			fmt.Sprintf(
				"kemono error %d: invalid creator URL found for kemono party: %s",
				utils.INPUT_ERROR,
				outlier,
			),
		)
		os.Exit(1)
	}

	valid, outlier = utils.SliceMatchesRegex(POST_URL_REGEX, k.PostUrls)
	if !valid {
		color.Red(
			fmt.Sprintf(
				"kemono error %d: invalid post URL found for kemono party: %s",
				utils.INPUT_ERROR,
				outlier,
			),
		)
		os.Exit(1)
	}

	if len(k.CreatorUrls) > 0 {
		if len(k.CreatorPageNums) == 0 {
			k.CreatorPageNums = make([]string, len(k.CreatorUrls))
		} else {
			utils.ValidatePageNumInput(
				len(k.CreatorUrls),
				k.CreatorPageNums,
				[]string{
					"Number of creator URL(s) and page numbers must be equal.",
				},
			)
		}
		creatorsToDl := ProcessCreatorUrls(k.CreatorUrls, k.CreatorPageNums)
		k.CreatorsToDl = append(k.CreatorsToDl, creatorsToDl...)
		k.CreatorUrls = nil
		k.CreatorPageNums = nil
	}
	if len(k.PostUrls) > 0 {
		postsToDl := ProcessPostUrls(k.PostUrls)
		k.PostsToDl = append(k.PostsToDl, postsToDl...)
		k.PostUrls = nil
	}
	k.RemoveDuplicates()
}

// KemonoDlOptions is the struct that contains the arguments for Kemono download options.
type KemonoDlOptions struct {
	DlAttachments bool
	DlGdrive      bool

	Configs       *configs.Config

	// GdriveClient is the Google Drive client to be
	// used in the download process for Pixiv Fanbox posts
	GdriveClient *gdrive.GDrive

	SessionCookieId string
	SessionCookies  []*http.Cookie
}

// ValidateArgs validates the session cookie ID of the Kemono account to download from.
// It also validates the Google Drive client if the user wants to download to Google Drive.
//
// Should be called after initialising the struct.
func (k *KemonoDlOptions) ValidateArgs(userAgent string) {
	if k.SessionCookieId != "" {
		k.SessionCookies = []*http.Cookie{
			api.VerifyAndGetCookie(utils.KEMONO, k.SessionCookieId, userAgent),
		}
	} else {
		color.Red("kemono error %d: session cookie ID is required", utils.INPUT_ERROR)
		os.Exit(1)
	}

	if k.DlGdrive && k.GdriveClient == nil {
		k.DlGdrive = false
	} else if !k.DlGdrive && k.GdriveClient != nil {
		k.GdriveClient = nil
	}
}
