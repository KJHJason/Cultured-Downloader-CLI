package models

type FantiaPost struct {
	Post struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		Thumb struct {
			Original string `json:"original"`
		} `json:"thumb"`
		Fanclub struct {
			User struct {
				Name string `json:"name"`
			} `json:"user"`
		} `json:"fanclub"`
		Status       string `json:"status"`
		PostContents []struct {
			// Any attachments such as pdfs that are on their dedicated section
			AttachmentURI string `json:"attachment_uri"`

			// For images that are uploaded to their own section
			PostContentPhotos []struct {
				ID  int `json:"id"`
				URL struct {
					Original string `json:"original"`
				} `json:"url"`
			} `json:"post_content_photos"`

			// For images that are embedded in the post content
			Comment string `json:"comment"`

			// for attachments such as pdfs that are embedded in the post content
			DownloadUri string `json:"download_uri"`
			Filename    string `json:"filename"`
		} `json:"post_contents"`
	} `json:"post"`
}
