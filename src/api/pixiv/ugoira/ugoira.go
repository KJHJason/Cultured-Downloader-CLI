package ugoira

import (
	"fmt"
	"os"
	"strings"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
	"github.com/fatih/color"
)

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
				"pixiv error %d: Output extension %q is not allowed for ugoira conversion",
				utils.INPUT_ERROR,
				u.OutputFormat,
			),
		},
	)
}
