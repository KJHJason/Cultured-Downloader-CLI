package pixivweb

import (
	"time"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

const (
	ILLUST = iota
	MANGA
	UGOIRA
)

// This is due to Pixiv's strict rate limiting.
//
// Without delays, the user might get 429 too many requests
// or the user's account might get suspended.
//
// Additionally, pixiv.net is protected by cloudflare, so
// to prevent the user's IP reputation from going down, delays are added.
//
// More info: https://github.com/Nandaka/PixivUtil2/issues/477
func pixivSleep() {
	time.Sleep(utils.GetRandomTime(0.5, 1.0))
}
