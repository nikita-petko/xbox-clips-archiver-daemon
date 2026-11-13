package models

import "time"

// Media is a generic representation of media content returned by the Xbox API, such as screenshots or game clips.
type Media struct {
	// Id is the unique identifier of the media content.
	Id string

	// TitleId is the identifier of the game title associated with the media content.
	TitleId int

	// TitleName is the name of the game title associated with the media content.
	TitleName string

	// CreatedAt is the timestamp when the media content was created.
	CreatedAt *time.Time

	// MediaUris is a list of download URIs for the media content.
	MediaUris []*DownloadURI
}
