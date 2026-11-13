package models

// DownloadURI represents the structure of a download URI returned by the Xbox API.
type DownloadURI struct {
	// URI is the actual download link for the content.
	URI string `json:"uri"`

	// FileSize is the size of the content in bytes.
	//
	// Used to compute the total size of the content when downloading multiple files
	// for progress tracking and validation purposes.
	FileSize int `json:"fileSize"`
}
