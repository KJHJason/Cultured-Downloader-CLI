package cmds

import (
	"bufio"
	"os"
	"io"
	"fmt"
	"strings"
	"regexp"

	"github.com/fatih/color"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const (
	PAGE_NUM_REGEX_GRP_NAME = "pageNum"

	// Pixiv Fanbox
	PF_BASE_REGEX_STR = `https://(?:www\.fanbox\.cc/@(?P<creatorId1>[\w.-]+)|(?P<creatorId2>[\w.-]+)\.fanbox\.cc)`

	// Pixiv
	P_BASE_REGEX_STR = `https://www\.pixiv\.net/(?:en/)?`
)

var (
	PAGE_NUM_REGEX_STR = fmt.Sprintf(
		`(?:; (?P<%s>[1-9]\d*(?:-[1-9]\d*)?))?`,
		PAGE_NUM_REGEX_GRP_NAME,
	)

	// Fantia
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

	// Pixiv Fanbox
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

	// Pixiv
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

// openTextFile opens the text file at the given path and returns a os.File and a bufio.Reader.
//
// If an error occurs, the program will exit with an error message and status code 1.
func openTextFile(textFilePath, website string) (*os.File, *bufio.Reader) {
	f, err := os.Open(textFilePath)
	if err != nil {
		errMsg := fmt.Sprintf(
			"error %d: failed to open %s cookie file at %s, more info => %v",
			utils.OS_ERROR,
			website,
			textFilePath,
			err,
		)
		color.Red(errMsg)
		os.Exit(1)
	}
	return f, bufio.NewReader(f)
}

// readLine reads a line from the given reader and returns the line as a slice of bytes.
//
// If the reader reaches EOF, the second return value will be true. Otherwise, it will be false.
// However, if an error occurs, the program will exit with an error message and status code 1.
func readLine(reader *bufio.Reader, textFilePath, website string) ([]byte, bool) {
	lineBytes, err := utils.ReadLine(reader)
	if err != nil {
		if err == io.EOF {
			return nil, true
		}
		errMsg := fmt.Sprintf(
			"error %d: failed to read fantia text file at %s, more info => %v",
			utils.OS_ERROR,
			textFilePath,
			err,
		)
		color.Red(errMsg)
		os.Exit(1)
	}
	return lineBytes, false
}

type parsedFantiaFanclub struct {
	FanclubId string
	PageNum   string
}

// parseFantiaTextFile parses the text file at the given path and returns a slice of post IDs and a slice of parsedFantiaFanclub.
func parseFantiaTextFile(textFilePath string) ([]string, []*parsedFantiaFanclub) {
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

type parsedPixivFanboxCreator struct {
	CreatorId string
	PageNum   string
}

// parsePixivFanboxTextFile parses the text file at the given path and returns a slice of post IDs and a slice of parsedPixivFanboxCreator.
func parsePixivFanboxTextFile(textFilePath string) ([]string, []*parsedPixivFanboxCreator) {
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

type parsedPixivArtist struct {
	ArtistId string
	PageNum  string
}
type parsedPixivTag struct {
	Tag      string
	PageNum  string
}

// parsePixivTextFile parses the text file at the given path and returns a slice of post IDs, a slice of parsedPixivArtist, and a slice of parsedPixivTag.
func parsePixivTextFile(textFilePath string) ([]string, []*parsedPixivArtist, []*parsedPixivTag) {
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
