package models

type CreatorPaginatedPostsJson struct {
	Body []string `json:"body"`
}

type FanboxCreatorPostsJson struct {
	Body struct {
		Items []struct {
			Id string `json:"id"`
		} `json:"items"`
	} `json:"body"`
}

type FanboxPostJson struct {
	Body struct {
		Id            string      `json:"id"`
		Title         string      `json:"title"`
		Type          string      `json:"type"`
		CreatorId     string      `json:"creatorId"`
		CoverImageUrl string      `json:"coverImageUrl"`
		Body          interface{} `json:"body"`
	} `json:"body"`
}

type FanboxFilePostJson struct {
	Text  string `json:"text"`
	Files []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Extension string `json:"extension"`
		Size      int    `json:"size"`
		Url       string `json:"url"`
	} `json:"files"`
}

type FanboxImagePostJson struct {
	Text   string `json:"text"`
	Images []struct {
		ID           string `json:"id"`
		Extension    string `json:"extension"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		OriginalUrl  string `json:"originalUrl"`
		ThumbnailUrl string `json:"thumbnailUrl"`
	} `json:"images"`
}

type FanboxTextPostJson struct {
	Text string `json:"text"`
}

type FanboxArticleBlocks []struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	ImageID string `json:"imageId,omitempty"`
	Styles  []struct {
		Type   string `json:"type"`
		Offset int    `json:"offset"`
		Length int    `json:"length"`
	} `json:"styles,omitempty"`
	Links []struct {
		Offset int    `json:"offset"`
		Length int    `json:"length"`
		Url    string `json:"url"`
	} `json:"links,omitempty"`
	FileID string `json:"fileId,omitempty"`
} 

type FanboxArticleJson struct {
	Blocks FanboxArticleBlocks `json:"blocks"`
	ImageMap map[string]struct {
		ID           string `json:"id"`
		Extension    string `json:"extension"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		OriginalUrl  string `json:"originalUrl"`
		ThumbnailUrl string `json:"thumbnailUrl"`
	} `json:"imageMap"`
	FileMap map[string]struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Extension string `json:"extension"`
		Size      int    `json:"size"`
		Url       string `json:"url"`
	} `json:"fileMap"`
}
