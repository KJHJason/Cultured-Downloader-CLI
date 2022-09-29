package utils

import (
	"runtime"
)

func GetUserAgent() string {
	var userAgent = map[string]string {
		"linux":
			"Mozilla/5.0 (X11; Linux x86_64)",
		"darwin":
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 12_6)",
	}
	userAgentOS := userAgent[runtime.GOOS]
	if (userAgentOS == "") {
		userAgentOS = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	}
	return userAgentOS + " AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36"
}

var (
	USER_AGENT = GetUserAgent()
)

const (
	RETRY_COUNTER = 5
	MAX_CONCURRENT_DOWNLOADS = 5
)