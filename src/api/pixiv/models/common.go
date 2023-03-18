package models

type Ugoira struct {
	Url      string
	FilePath string
	Frames   map[string]int64
}

type UgoiraFramesJson []*struct {
	File string `json:"file"`
	Delay float64 `json:"delay"`
}
