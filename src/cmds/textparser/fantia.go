package textparser

import (
	"fmt"
	"strings"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

var (
	F_POST_URL_REGEX = regexp.MustCompile(
		`^https://fantia\.jp/posts/(?P<postId>\d+)$`,
	)
	F_POST_REGEX_POST_ID_INDEX = F_POST_URL_REGEX.SubexpIndex("postId")
	F_FANCLUB_URL_REGEX = regexp.MustCompile(
		// ^https://fantia\.jp/fanclubs/(?P<fanclubId>\d+)(?:/posts)?(?:; (?P<pageNum>[1-9]\d*(?:-[1-9]\d*)?))?$
		fmt.Sprintf(
			`^https://fantia\.jp/fanclubs/(?P<fanclubId>\d+)(?:/posts)?%s$`,
			PAGE_NUM_REGEX_STR,
		),
	)
	F_FANCLUB_REGEX_FANCLUB_ID_INDEX = F_FANCLUB_URL_REGEX.SubexpIndex("fanclubId")
	F_FANCLUB_REGEX_PAGE_NUM_INDEX = F_FANCLUB_URL_REGEX.SubexpIndex(PAGE_NUM_REGEX_GRP_NAME)
)

type parsedFantiaFanclub struct {
	FanclubId string
	PageNum   string
}

// parseFantiaTextFile parses the text file at the given path and returns a slice of post IDs and a slice of parsedFantiaFanclub.
func ParseFantiaTextFile(textFilePath string) ([]string, []*parsedFantiaFanclub) {
	f, reader := openTextFile(
		textFilePath, 
		utils.FANTIA,
	)
	defer f.Close() 

	var postIds []string
	var fanclubIds []*parsedFantiaFanclub
	for {
		lineBytes, isEof := readLine(reader, textFilePath, utils.FANTIA)
		if isEof {
			break
		}

		url := strings.TrimSpace(string(lineBytes))
		if url == "" {
			continue
		}

		if matched := F_POST_URL_REGEX.FindStringSubmatch(url); matched != nil {
			postIds = append(postIds, matched[F_POST_REGEX_POST_ID_INDEX])
			continue
		}

		if matched := F_FANCLUB_URL_REGEX.FindStringSubmatch(url); matched != nil {
			fanclubIds = append(fanclubIds, &parsedFantiaFanclub{
				FanclubId: matched[F_FANCLUB_REGEX_FANCLUB_ID_INDEX],
				PageNum:   matched[F_FANCLUB_REGEX_PAGE_NUM_INDEX],
			})
			continue
		}
	}

	return postIds, fanclubIds
}
