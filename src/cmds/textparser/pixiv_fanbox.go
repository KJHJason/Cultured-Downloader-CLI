package textparser

import (
	"fmt"
	"strings"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const PF_BASE_REGEX_STR = `https://(?:www\.fanbox\.cc/@(?P<creatorId1>[\w.-]+)|(?P<creatorId2>[\w.-]+)\.fanbox\.cc)`

var (
	PF_POST_URL_REGEX = regexp.MustCompile(
		// ^https://(?:www\.fanbox\.cc/@(?P<creatorId1>[\w.-]+)|(?P<creatorId2>[\w.-]+)\.fanbox\.cc)/posts/(?P<postId>\d+)$
		fmt.Sprintf(
			`^%s/posts/(?P<postId>\d+)$`,
			PF_BASE_REGEX_STR,
		),
	)
	PF_POST_REGEX_POST_ID_INDEX = PF_POST_URL_REGEX.SubexpIndex("postId")
	PF_CREATOR_URL_REGEX = regexp.MustCompile(
		// ^https://(?:www\.fanbox\.cc/@(?P<creatorId1>[\w.-]+)|(?P<creatorId2>[\w.-]+)\.fanbox\.cc)(?:/posts)?(?:; (?P<pageNum>[1-9]\d*(?:-[1-9]\d*)?))?$
		fmt.Sprintf(
			`^%s(?:/posts)?%s$`,
			PF_BASE_REGEX_STR,
			PAGE_NUM_REGEX_STR,
		),
	)
	PF_CREATOR_REGEX_CREATOR_ID_INDEX_1 = PF_CREATOR_URL_REGEX.SubexpIndex("creatorId1")
	PF_CREATOR_REGEX_CREATOR_ID_INDEX_2 = PF_CREATOR_URL_REGEX.SubexpIndex("creatorId2")
	PF_CREATOR_REGEX_PAGE_NUM_INDEX = PF_CREATOR_URL_REGEX.SubexpIndex(PAGE_NUM_REGEX_GRP_NAME)
)

type parsedPixivFanboxCreator struct {
	CreatorId string
	PageNum   string
}

// ParsePixivFanboxTextFile parses the text file at the given path and returns a slice of post IDs and a slice of parsedPixivFanboxCreator.
func ParsePixivFanboxTextFile(textFilePath string) ([]string, []*parsedPixivFanboxCreator) {
	lowercaseFanbox := strings.ToLower(utils.PIXIV_FANBOX_TITLE)
	f, reader := openTextFile(
		textFilePath, 
		lowercaseFanbox,
	)
	defer f.Close()

	var postIds []string
	var creatorIds []*parsedPixivFanboxCreator
	for {
		lineBytes, isEof := readLine(reader, textFilePath, lowercaseFanbox)
		if isEof {
			break
		}

		url := strings.TrimSpace(string(lineBytes))
		if url == "" {
			continue
		}

		if matched := PF_POST_URL_REGEX.FindStringSubmatch(url); matched != nil {
			postIds = append(postIds, matched[PF_POST_REGEX_POST_ID_INDEX])
			continue
		}

		if matched := PF_CREATOR_URL_REGEX.FindStringSubmatch(url); matched != nil {
			creatorId := matched[PF_CREATOR_REGEX_CREATOR_ID_INDEX_1]
			if creatorId == "" {
				creatorId = matched[PF_CREATOR_REGEX_CREATOR_ID_INDEX_2]
			}

			creatorIds = append(creatorIds, &parsedPixivFanboxCreator{
				CreatorId: creatorId,
				PageNum:   matched[PF_CREATOR_REGEX_PAGE_NUM_INDEX],
			})
			continue
		}
	}

	return postIds, creatorIds
}
