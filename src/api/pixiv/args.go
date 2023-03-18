package pixiv

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

// PixivDl contains the IDs of the Pixiv artworks and
// illustrators and Tag Names to download.
type PixivDl struct {
	ArtworkIds []string

	IllustratorIds      []string
	IllustratorPageNums []string

	TagNames         []string
	TagNamesPageNums []string
}

// ValidateArgs validates the IDs of the Pixiv artworks and illustrators to download.
//
// It also validates the page numbers of the tag names to download.
//
// Should be called after initialising the struct.
func (p *PixivDl) ValidateArgs() {
	utils.ValidateIds(p.ArtworkIds)
	utils.ValidateIds(p.IllustratorIds)
	p.ArtworkIds = utils.RemoveSliceDuplicates(p.ArtworkIds)

	if len(p.IllustratorPageNums) > 0 {
		utils.ValidatePageNumInput(
			len(p.IllustratorIds),
			p.IllustratorPageNums,
			[]string{
				"Number of illustrators ID(s) and illustrators' page numbers must be equal.",
			},
		)
	} else {
		p.IllustratorPageNums = make([]string, len(p.IllustratorIds))
	}
	p.IllustratorIds, p.IllustratorPageNums = utils.RemoveDuplicateIdAndPageNum(
		p.IllustratorIds,
		p.IllustratorPageNums,
	)

	if len(p.TagNamesPageNums) > 0 {
		utils.ValidatePageNumInput(
			len(p.TagNames),
			p.TagNamesPageNums,
			[]string{
				"Number of tag names and tag names' page numbers must be equal.",
			},
		)
	} else {
		p.TagNamesPageNums = make([]string, len(p.TagNames))
	}
	p.TagNames, p.TagNamesPageNums = utils.RemoveDuplicateIdAndPageNum(
		p.TagNames,
		p.TagNamesPageNums,
	)
}

// UgoiraDlOptions is the struct that contains the
// configs for the processing of the ugoira images after downloading from Pixiv.
type UgoiraOptions struct {
	DeleteZip    bool
	Quality      int
	OutputFormat string
}

var UGOIRA_ACCEPTED_EXT = []string{
	".gif",
	".apng",
	".webp",
	".webm",
	".mp4",
}

// ValidateArgs validates the arguments of the ugoira process options.
//
// Should be called after initialising the struct.
func (u *UgoiraOptions) ValidateArgs() {
	u.OutputFormat = strings.ToLower(u.OutputFormat)

	// u.Quality is only for .mp4 and .webm
	if u.OutputFormat == ".mp4" && u.Quality < 0 || u.Quality > 51 {
		color.Red(
			fmt.Sprintf(
				"pixiv error %d: Ugoira quality of %d is not allowed",
				utils.INPUT_ERROR,
				u.Quality,
			),
		)
		color.Red("Ugoira quality for FFmpeg must be between 0 and 51 for .mp4")
		os.Exit(1)
	} else if u.OutputFormat == ".webm" && u.Quality < 0 || u.Quality > 63 {
		color.Red(
			fmt.Sprintf(
				"pixiv error %d: Ugoira quality of %d is not allowed",
				utils.INPUT_ERROR,
				u.Quality,
			),
		)
		color.Red("Ugoira quality for FFmpeg must be between 0 and 63 for .webm")
		os.Exit(1)
	}

	u.OutputFormat = strings.ToLower(u.OutputFormat)
	utils.ValidateStrArgs(
		u.OutputFormat,
		UGOIRA_ACCEPTED_EXT,
		[]string{
			fmt.Sprintf(
				"pixiv error %d: Output extension \"%s\" is not allowed for ugoira conversion",
				utils.INPUT_ERROR,
				u.OutputFormat,
			),
		},
	)
}

// PixivToDl is the struct that contains the arguments of Pixiv download options.
type PixivDlOptions struct {
	// Sort order of the results. Can be "date_desc" or "date_asc".
	SortOrder   string
	SearchMode  string
	RatingMode  string
	ArtworkType string

	MobileClient *PixivMobile
	RefreshToken string

	SessionCookies  []*http.Cookie
	SessionCookieId string
}

var (
	ACCEPTED_SORT_ORDER = []string{
		"date", "date_d",
		"popular", "popular_d",
		"popular_male", "popular_male_d",
		"popular_female", "popular_female_d",
	}
	ACCEPTED_SEARCH_MODE = []string{
		"s_tag",
		"s_tag_full",
		"s_tc",
	}
	ACCEPTED_RATING_MODE = []string{
		"safe",
		"r18",
		"all",
	}
	ACCEPTED_ARTWORK_TYPE = []string{
		"illust_and_ugoira",
		"manga",
		"all",
	}
)

// ValidateArgs validates the arguments of the Pixiv download options.
//
// Should be called after initialising the struct.
func (p *PixivDlOptions) ValidateArgs(userAgent string) {
	p.SortOrder = strings.ToLower(p.SortOrder)
	utils.ValidateStrArgs(
		p.SortOrder,
		ACCEPTED_SORT_ORDER,
		[]string{
			fmt.Sprintf(
				"pixiv error %d: Sort order %s is not allowed",
				utils.INPUT_ERROR,
				p.SortOrder,
			),
		},
	)

	p.SearchMode = strings.ToLower(p.SearchMode)
	utils.ValidateStrArgs(
		p.SearchMode,
		ACCEPTED_SEARCH_MODE,
		[]string{
			fmt.Sprintf(
				"pixiv error %d: Search order %s is not allowed",
				utils.INPUT_ERROR,
				p.SearchMode,
			),
		},
	)

	p.RatingMode = strings.ToLower(p.RatingMode)
	utils.ValidateStrArgs(
		p.RatingMode,
		ACCEPTED_RATING_MODE,
		[]string{
			fmt.Sprintf(
				"pixiv error %d: Rating order %s is not allowed",
				utils.INPUT_ERROR,
				p.RatingMode,
			),
		},
	)

	p.ArtworkType = strings.ToLower(p.ArtworkType)
	utils.ValidateStrArgs(
		p.ArtworkType,
		ACCEPTED_ARTWORK_TYPE,
		[]string{
			fmt.Sprintf(
				"pixiv error %d: Artwork type %s is not allowed",
				utils.INPUT_ERROR,
				p.ArtworkType,
			),
		},
	)

	if p.SessionCookieId != "" {
		p.SessionCookies = []*http.Cookie{
			api.VerifyAndGetCookie(utils.PIXIV, p.SessionCookieId, userAgent),
		}
	}

	if p.RefreshToken != "" {
		p.MobileClient = NewPixivMobile(p.RefreshToken, 10)
		if p.RatingMode != "all" {
			color.Red(
				utils.CombineStringsWithNewline(
					[]string{
						fmt.Sprintf(
							"pixiv error %d: when using the refresh token, only \"all\" is supported for the --rating_mode flag.",
							utils.INPUT_ERROR,
						),
						fmt.Sprintf(
							"hence, the rating mode will be updated from \"%s\" to \"all\"...\n",
							p.RatingMode,
						),
					},
				),
			)
			p.RatingMode = "all"
		}

		if p.ArtworkType == "illust_and_ugoira" {
			// convert "illust_and_ugoira" to "illust"
			// since the mobile API does not support "illust_and_ugoira"
			// However, there will still be ugoira posts in the results
			p.ArtworkType = "illust"
		}

		// Convert search mode to the correct value
		// based on the Pixiv's ajax web API
		switch p.SearchMode {
		case "s_tag":
			p.SearchMode = "partial_match_for_tags"
		case "s_tag_full":
			p.SearchMode = "exact_match_for_tags"
		case "s_tc":
			p.SearchMode = "title_and_caption"
		default:
			panic(
				fmt.Sprintf(
					"pixiv mobile error %d: invalid search mode \"%s\"",
					utils.DEV_ERROR,
					p.SearchMode,
				),
			)
		}

		// Convert sort order to the correct value
		// based on the Pixiv's ajax web API
		var newSortOrder string
		if strings.Contains(p.SortOrder, "popular") {
			newSortOrder = "popular_desc" // only supports popular_desc
		} else if p.SortOrder == "date_d" {
			newSortOrder = "date_desc"
		} else {
			newSortOrder = "date_asc"
		}

		if p.SortOrder != "date" && p.SortOrder != "date_d" && p.SortOrder != "popular_d" {
			var ajaxEquivalent string
			switch newSortOrder {
			case "popular_desc":
				ajaxEquivalent = "popular_d"
			case "date_desc":
				ajaxEquivalent = "date_d"
			case "date_asc":
				ajaxEquivalent = "date"
			default:
				panic(
					fmt.Sprintf(
						"pixiv error %d: unknown sort order \"%s\" in PixivDlOptions.ValidateArgs()",
						utils.DEV_ERROR,
						newSortOrder,
					),
				)
			}

			color.Red(
				utils.CombineStringsWithNewline(
					[]string{
						fmt.Sprintf(
							"pixiv error %d: when using the refresh token, only \"date\", \"date_d\", \"popular_d\" are supported for the --sort_order flag.",
							utils.INPUT_ERROR,
						),
						fmt.Sprintf(
							"hence, the sort order will be updated from \"%s\" to \"%s\"...\n",
							p.SortOrder,
							ajaxEquivalent,
						),
					},
				),
			)
		}
		p.SortOrder = newSortOrder
	}
}
