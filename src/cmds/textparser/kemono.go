package textparser

import (
	"strings"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono"
	"github.com/KJHJason/Cultured-Downloader-CLI/api/kemono/models"
)

var (
	K_POST_URL_REGEX = regexp.MustCompile(kemono.BASE_REGEX_STR + kemono.BASE_POST_SUFFIX_REGEX_STR)
	K_POST_REGEX_SERVICE_INDEX = K_POST_URL_REGEX.SubexpIndex(kemono.SERVICE_GROUP_NAME)
	K_POST_REGEX_CREATOR_ID_INDEX = K_POST_URL_REGEX.SubexpIndex(kemono.CREATOR_ID_GROUP_NAME)
	K_POST_REGEX_POST_ID_INDEX = K_POST_URL_REGEX.SubexpIndex(kemono.POST_ID_GROUP_NAME)

	K_CREATOR_URL_REGEX = regexp.MustCompile(kemono.BASE_REGEX_STR + PAGE_NUM_REGEX_STR)
	K_CREATOR_REGEX_CREATOR_ID_INDEX = K_CREATOR_URL_REGEX.SubexpIndex(kemono.CREATOR_ID_GROUP_NAME)
	K_CREATOR_REGEX_PAGE_NUM_INDEX = K_CREATOR_URL_REGEX.SubexpIndex(PAGE_NUM_REGEX_GRP_NAME)
)

// ParseKemonoTextFile parses the text file at the given path and returns a slice of KemonoPostToDl and a slice of KemonoCreatorToDl.
func ParseKemonoTextFile(textFilePath string) ([]*models.KemonoPostToDl, []*models.KemonoCreatorToDl) {
	lowercaseFanbox := strings.ToLower(utils.PIXIV_FANBOX_TITLE)
	f, reader := openTextFile(
		textFilePath, 
		lowercaseFanbox,
	)
	defer f.Close()

	var postsToDl []*models.KemonoPostToDl
	var creatorsToDl []*models.KemonoCreatorToDl
	for {
		lineBytes, isEof := readLine(reader, textFilePath, lowercaseFanbox)
		if isEof {
			break
		}

		url := strings.TrimSpace(string(lineBytes))
		if url == "" {
			continue
		}

		if matched := K_POST_URL_REGEX.FindStringSubmatch(url); matched != nil {
			postsToDl = append(postsToDl, &models.KemonoPostToDl{
				Service: matched[K_POST_REGEX_SERVICE_INDEX],
				CreatorId: matched[K_POST_REGEX_CREATOR_ID_INDEX],
				PostId: matched[K_POST_REGEX_POST_ID_INDEX],
			})
			continue
		}

		if matched := K_CREATOR_URL_REGEX.FindStringSubmatch(url); matched != nil {
			creatorsToDl = append(creatorsToDl, &models.KemonoCreatorToDl{
				Service: matched[K_POST_REGEX_SERVICE_INDEX],
				CreatorId: matched[K_CREATOR_REGEX_CREATOR_ID_INDEX],
				PageNum: matched[K_CREATOR_REGEX_PAGE_NUM_INDEX],
			})
			continue
		}
	}

	return postsToDl, creatorsToDl
}
