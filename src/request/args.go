package request

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"regexp"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

type RequestArgs struct {
	// Main Request Options
	Method string
	Url string
	Timeout int

	// Additional Request Options
	Headers            map[string]string
	Params             map[string]string
	Cookies            []*http.Cookie
	UserAgent          string
	DisableCompression bool

	// HTTP/2 and HTTP/3 Options
	Http2 bool
	Http3 bool

	// Check status will check the status code of the response for 200 OK.
	// If the status code is not 200 OK, it will retry several times and 
	// if the status code is still not 200 OK, it will return an error.
	// Otherwise, it will return the response regardless of the status code.
	CheckStatus bool

	// Context is used to cancel the request if needed.
	// E.g. if the user presses Ctrl+C, we can use context.WithCancel(context.Background())
	Context context.Context
}

var (
	// Since the URLs below will be redirected to Fantia's AWS S3 URL, 
	// we need to use HTTP/2 as it is not supported by HTTP/3 yet.
	FANTIA_ALBUM_URL = regexp.MustCompile(
		`^https://fantia.jp/posts/[\d]+/album_image`,
	)
	FANTIA_DOWNLOAD_URL = regexp.MustCompile(
		`^https://fantia.jp/posts/[\d]+/download/[\d]+`,
	)

	HTTP3_SUPPORT_ARR = [...]string{
		"https://www.pixiv.net",
		"https://app-api.pixiv.net",

		"https://www.google.com",
		"https://drive.google.com",
	}
)

// ValidateArgs validates the arguments of the request
//
// Will panic if the arguments are invalid as this is a developer error
func (args *RequestArgs) ValidateArgs() {
	if args.Method == "" {
		panic(
			fmt.Errorf(
				"error %d: method cannot be empty",
				utils.DEV_ERROR,
			),
		)
	}

	if args.Headers == nil {
		args.Headers = make(map[string]string)
	}

	if args.Params == nil {
		args.Params = make(map[string]string)
	}

	if args.Cookies == nil {
		args.Cookies = make([]*http.Cookie, 0)
	}

	if args.UserAgent == "" {
		args.UserAgent = utils.USER_AGENT
	}

	if args.Context == nil {
		args.Context = context.Background()
	}

	if !args.Http2 && !args.Http3 {
		// if http2 and http3 are not enabled,
		// do a check to determine which protocol to use.
		if FANTIA_DOWNLOAD_URL.MatchString(args.Url) || FANTIA_ALBUM_URL.MatchString(args.Url) {
			args.Http2 = true
		} else {
			// check if the URL supports HTTP/3 first
			// before falling back to the default HTTP/2.
			for _, domain := range HTTP3_SUPPORT_ARR {
				if strings.HasPrefix(args.Url, domain) {
					args.Http3 = true
					break
				}
			}
			// if HTTP/3 is not supported, fall back to HTTP/2
			if !args.Http3 {
				args.Http2 = true
			}
		}
	} else if args.Http2 && args.Http3 {
		panic(
			fmt.Errorf(
				"error %d: http2 and http3 cannot be enabled at the same time",
				utils.DEV_ERROR,
			),
		)
	}

	if args.Url == "" {
		panic(
			fmt.Errorf(
				"error %d: url cannot be empty",
				utils.DEV_ERROR,
			),
		)
	}

	if args.Timeout < 0 {
		panic(
			fmt.Errorf(
				"error %d: timeout cannot be negative",
				utils.DEV_ERROR,
			),
		)
	} else if args.Timeout == 0 {
		args.Timeout = 15
	}
}
