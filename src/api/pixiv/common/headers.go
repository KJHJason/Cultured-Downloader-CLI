package pixivcommon

import (
	"fmt"

	"github.com/KJHJason/Cultured-Downloader-CLI/utils"
)

// Returns a defined request header needed to communicate with Pixiv's API
func GetPixivRequestHeaders() map[string]string {
	return map[string]string{
		"Origin":  utils.PIXIV_URL,
		"Referer": utils.PIXIV_URL,
	}
}

// Get the Pixiv illust page URL for the referral header value
func GetIllustUrl(illustId string) string {
	return fmt.Sprintf(
		"%s/artworks/%s",
		utils.PIXIV_URL,
		illustId,
	)
}

// Get the Pixiv user page URL for the referral header value
func GetUserUrl(userId string) string {
	return fmt.Sprintf(
		"%s/users/%s",
		utils.PIXIV_URL,
		userId,
	)
}
