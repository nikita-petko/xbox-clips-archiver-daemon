package models

// Media is a generic representation of media content returned by the Xbox API, such as screenshots or game clips.
type Media struct {
	// Id is the unique identifier of the media content.
	Id string `json:"id"`

	// TitleId is the identifier of the game title associated with the media content.
	TitleId int `json:"titleId"`

	// TitleName is the name of the game title associated with the media content.
	TitleName string `json:"titleName"`

	// MediaUris is a list of download URIs for the media content.
	MediaUris []*DownloadURI `json:"mediaUris"`
}
