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
	ArtworkIds     []string
	IllustratorIds []string

	// Tag names download options
	TagNames         []string
	TagNamesPageNums []string
}

// ValidateArgs validates the IDs of the Pixiv artworks and illustrators to download.
//
// It also validates the page numbers of the tag names to download.
func (p *PixivDl) ValidateArgs() {
	utils.ValidateIds(&p.ArtworkIds)
	utils.ValidateIds(&p.IllustratorIds)

	utils.ValidatePageNumInput(
		len(p.TagNames),
		&p.TagNamesPageNums,
		[]string{
			"Number of tag names and tag name's page numbers must be equal.",
		},
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
func (u *UgoiraOptions) ValidateArgs() {
	if u.Quality < 0 || u.Quality > 63 {
		color.Red(
			fmt.Sprintf("Pixiv: Ugoira quality of %d is nto allowed", u.Quality),
		)
		color.Red("Ugoira quality for FFmpeg must be between 0 and 63")
		os.Exit(1)
	}

	utils.ValidateStrArgs(
		u.OutputFormat,
		UGOIRA_ACCEPTED_EXT,
		[]string{
			fmt.Sprintf("Pixiv: Output extension \"%s\" is not allowed for ugoira conversion", u.OutputFormat),
		},
	)
	u.OutputFormat = strings.ToLower(u.OutputFormat)
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

	SessionCookies  []http.Cookie
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
func (p *PixivDlOptions) ValidateArgs() {
	utils.ValidateStrArgs(
		p.SortOrder,
		ACCEPTED_SORT_ORDER,
		[]string{
			fmt.Sprintf("Pixiv: Sort order %v is not allowed", p.SortOrder),
		},
	)
	p.SortOrder = strings.ToLower(p.SortOrder)

	utils.ValidateStrArgs(
		p.SearchMode,
		ACCEPTED_SEARCH_MODE,
		[]string{
			fmt.Sprintf("Pixiv: Search order %v is not allowed", p.SearchMode),
		},
	)
	p.SearchMode = strings.ToLower(p.SearchMode)

	utils.ValidateStrArgs(
		p.RatingMode,
		ACCEPTED_RATING_MODE,
		[]string{
			fmt.Sprintf("Pixiv: Rating order %v is not allowed", p.RatingMode),
		},
	)
	p.RatingMode = strings.ToLower(p.RatingMode)

	utils.ValidateStrArgs(
		p.ArtworkType,
		ACCEPTED_ARTWORK_TYPE,
		[]string{
			fmt.Sprintf("Pixiv: Artwork type %v is not allowed", p.ArtworkType),
		},
	)
	p.ArtworkType = strings.ToLower(p.ArtworkType)

	if p.SessionCookieId != "" {
		p.SessionCookies = []http.Cookie{
			api.VerifyAndGetCookie(utils.PIXIV, p.SessionCookieId),
		}
	}

	if p.RefreshToken != "" {
		p.MobileClient = NewPixivMobile(p.RefreshToken, 10)
	}
}
