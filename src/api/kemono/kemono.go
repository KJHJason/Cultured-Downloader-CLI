package kemono

type KemonoRes []struct {
	Added       string `json:"added"`
	Attachments []struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"attachments"`
	Content string `json:"content"`
	Edited  string `json:"edited"`
	Embed   struct {
		// embed will be ignored regardless of its value
		Description string `json:"description"`
		Subject     string `json:"subject"`
		Url         string `json:"url"`
	} `json:"embed"`
	File struct {
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"file"`
	Id         string `json:"id"`
	Published  string `json:"published"`
	Service    string `json:"service"`
	SharedFile bool   `json:"shared_file"`
	Title      string `json:"title"`
	User       string `json:"user"`
}


