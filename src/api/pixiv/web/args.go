package pixivweb

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/api"
	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// PixivToDl is the struct that contains the arguments of Pixiv download options.
type PixivWebDlOptions struct {
	// Sort order of the results. Can be "date_desc" or "date_asc".
	SortOrder   string
	SearchMode  string
	RatingMode  string
	ArtworkType string

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
func (p *PixivWebDlOptions) ValidateArgs(userAgent string) {
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
}
