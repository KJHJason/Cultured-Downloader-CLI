package models

type PixivOauthJson struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   float64 `json:"expires_in"`
}

type PixivOauthFlowJson struct {
	RefreshToken string `json:"refresh_token"`
}

type UgoiraJson struct {
	Metadata struct {
		Frames UgoiraFramesJson `json:"frames"`
		ZipUrls struct {
			Medium string `json:"medium"`
		} `json:"zip_urls"`
	} `json:"ugoira_metadata"`
}

type PixivMobileIllustJson struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"`

	User struct {
		Name  string `json:"name"`
	} `json:"user"`

	MetaSinglePage struct {
		OriginalImageUrl string `json:"original_image_url"`
	} `json:"meta_single_page"`

	MetaPages []struct {
		ImageUrls struct {
			Original string `json:"original"`
		} `json:"image_urls"`
	} `json:"meta_pages"`
}

type PixivMobileArtworkJson struct {
	Illust *PixivMobileIllustJson `json:"illust"`
}
type PixivMobileArtworksJson struct {
	Illusts []*PixivMobileIllustJson `json:"illusts"`
	NextUrl *string                  `json:"next_url"`
}
