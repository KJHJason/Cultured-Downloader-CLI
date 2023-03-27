package models

type GDriveFile struct {
	Kind        string `json:"kind"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	Size        string `json:"size"`
	MimeType    string `json:"mimeType"`
	Md5Checksum string `json:"md5Checksum"`
}

type GDriveFolder struct {
	Kind             string       `json:"kind"`
	IncompleteSearch bool         `json:"incompleteSearch"`
	Files            []GDriveFile `json:"files"`
	NextPageToken    string       `json:"nextPageToken"`
}

type GDriveToDl struct {
	Id 	     string
	Type     string
	FilePath string
}

type GdriveFileToDl struct {
	Id          string
	Name        string
	Size        string
	MimeType    string
	Md5Checksum string
	FilePath    string
}

type GdriveError struct {
	Err      error
	FilePath string
}
