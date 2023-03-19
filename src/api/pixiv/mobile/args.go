package pixivmobile

import (
	"fmt"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

// PixivToDl is the struct that contains the arguments of Pixiv download options.
type PixivMobileDlOptions struct {
	// Sort order of the results. Can be "date_desc" or "date_asc".
	SortOrder   string
	SearchMode  string
	RatingMode  string
	ArtworkType string

	MobileClient *PixivMobile
	RefreshToken string
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
func (p *PixivMobileDlOptions) ValidateArgs(userAgent string) {
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
							"hence, the rating mode will be updated from %q to \"all\"...\n",
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
					"pixiv mobile error %d: invalid search mode %q",
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
						"pixiv error %d: unknown sort order %q in PixivDlOptions.ValidateArgs()",
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
							"hence, the sort order will be updated from %q to %q...\n",
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
