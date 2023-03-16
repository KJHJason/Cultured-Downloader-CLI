package kemono

import (
	"fmt"
	"os"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

const BASE_REGEX_STR = `https://kemono\.party/(?P<service>patreon|fanbox|gumroad|subscribestar|dlsite|fantia|boosty)/user/(?P<creatorId>[\w-]+)`
var (
	POST_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%s/post/(?P<postId>\d+)$`,
			BASE_REGEX_STR,
		),
	)
	CREATOR_URL_REGEX = regexp.MustCompile(
		fmt.Sprintf(
			`^%s$`,
			BASE_REGEX_STR,
		),
	)
)

type KemonoDl struct {
	CreatorUrls     []string
	CreatorPageNums []string

	PostUrls []string
}

func (k *KemonoDl) ValidateArgs() {
	utils.ValidatePageNumInput(
		len(k.CreatorUrls),
		k.CreatorPageNums,
		[]string{
			"Number of creator URL(s) and page numbers must be equal.",
		},
	)

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
}
