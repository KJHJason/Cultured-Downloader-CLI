package request

import "net/http"

type ToDownload struct {
	Url      string
	FilePath string
}

type DlOptions struct {
	// MaxConcurrency is the maximum number of concurrent downloads
	MaxConcurrency int

	// Cookies is a list of cookies to be used in the download process
	Cookies []*http.Cookie

	// Headers is a map of headers to be used in the download process
	Headers map[string]string

	// UseHttp3 is a flag to enable HTTP/3
	// Otherwise, HTTP/2 will be used by default
	UseHttp3 bool
}
