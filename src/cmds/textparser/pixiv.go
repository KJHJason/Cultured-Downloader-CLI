package textparser

import (
	"fmt"
	"strings"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const P_BASE_REGEX_STR = `https://www\.pixiv\.net/(?:en/)?`

var (
	P_ILLUST_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%sartworks/(?P<illustId>\d+)$`,
			P_BASE_REGEX_STR,
		),
	)
	P_ILLUST_REGEX_ID_INDEX = P_ILLUST_URL_REGEX.SubexpIndex("illustId")
	P_ARTIST_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%susers/(?P<artistId>\d+)%s$`,
			P_BASE_REGEX_STR,
			PAGE_NUM_REGEX_STR,
		),
	)
	P_ARTIST_REGEX_ID_INDEX = P_ARTIST_URL_REGEX.SubexpIndex("artistId")
	P_ARTIST_REGEX_PAGE_NUM_INDEX = P_ARTIST_URL_REGEX.SubexpIndex(PAGE_NUM_REGEX_GRP_NAME)
	P_TAG_URL_REGEX = regexp.MustCompile(
		// ^https://www\.pixiv\.net/(?:en/)?tags/(?P<tag>[\w-%()]+)(?:/(?:artworks|illustrations|manga))?(?:\?[\w=&-.]+)?(?:; (?P<pageNum>[1-9]\d*(?:-[1-9]\d*)?))?$
		"^" + P_BASE_REGEX_STR + `tags/(?P<tag>[\w-%()]+)(?:/(?:artworks|illustrations|manga))?(?:\?[\w=&-.]+)?` + PAGE_NUM_REGEX_STR + "$",
	)
	P_TAG_REGEX_TAG_INDEX = P_TAG_URL_REGEX.SubexpIndex("tag")
	P_TAG_REGEX_PAGE_NUM_INDEX = P_TAG_URL_REGEX.SubexpIndex(PAGE_NUM_REGEX_GRP_NAME)
)

type parsedPixivArtist struct {
	ArtistId string
	PageNum  string
}
type parsedPixivTag struct {
	Tag      string
	PageNum  string
}

// ParsePixivTextFile parses the text file at the given path and returns a slice of post IDs, a slice of parsedPixivArtist, and a slice of parsedPixivTag.
func ParsePixivTextFile(textFilePath string) ([]string, []*parsedPixivArtist, []*parsedPixivTag) {
	f, reader := openTextFile(
		textFilePath, 
		utils.PIXIV,
	)
	defer f.Close()

	var postIds []string
	var artistIds []*parsedPixivArtist
	var tags []*parsedPixivTag
	for {
		lineBytes, isEof := readLine(reader, textFilePath, utils.PIXIV)
		if isEof {
			break
		}

		url := strings.TrimSpace(string(lineBytes))
		if url == "" {
			continue
		}

		if matched := P_ILLUST_URL_REGEX.FindStringSubmatch(url); matched != nil {
			postIds = append(postIds, matched[P_ILLUST_REGEX_ID_INDEX])
			continue
		}

		if matched := P_ARTIST_URL_REGEX.FindStringSubmatch(url); matched != nil {
			artistIds = append(artistIds, &parsedPixivArtist{
				ArtistId: matched[P_ARTIST_REGEX_ID_INDEX],
				PageNum:  matched[P_ARTIST_REGEX_PAGE_NUM_INDEX],
			})
			continue
		}

		if matched := P_TAG_URL_REGEX.FindStringSubmatch(url); matched != nil {
			tags = append(tags, &parsedPixivTag{
				Tag:      matched[P_TAG_REGEX_TAG_INDEX],
				PageNum:  matched[P_TAG_REGEX_PAGE_NUM_INDEX],
			})
			continue
		}
	}

	return postIds, artistIds, tags
}
