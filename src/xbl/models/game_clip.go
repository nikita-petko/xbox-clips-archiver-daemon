package models

// GameClip represents the structure of a game clip returned by the Xbox API
// filled in with only required information
type GameClip struct {
	// GameClipId is the unique identifier of the game clip.
	//
	// Used within the cache to ensure that
	// each game clip is uniquely identified and
	// isn't duplicated within the cache.
	GameClipId string `json:"gameClipId"`

	// DateRecorded is the timestamp when the game clip was taken.
	//
	// It serves as the name of the game clip on S3.
	DateRecorded string `json:"dateRecorded"`

	// TitleId is the identifier of the game title associated with the game clip.
	//
	// Only relevant if GroupByTitleId is true, otherwise TitleName is used.
	TitleId int `json:"titleId"`

	// TitleName is the name of the game title associated with the game clip.
	//
	// Only relevant if GroupByTitleId is false, otherwise TitleId is used.
	TitleName string `json:"titleName"`

	// GameClipUris is a list of download URIs for the game clip.
	GameClipUris []*DownloadURI `json:"gameClipUris"`
}

// GameClipContent represents the actual content of a game clip response returned by the Xbox API.
type GameClipContent struct {
	// GameClips is the list of game clips returned by the API.
	GameClips []*GameClip `json:"gameClips"`

	// PagingInfo contains the paging information for the game clip results.
	PagingInfo *PagingInfo `json:"pagingInfo"`
}

// GameClipResponse represents the response structure returned by the Xbox API for game clips.
type GameClipResponse struct {
	// Content contains the actual game clip content.
	Content *GameClipContent `json:"content"`

	// Code is the status code returned by the API.
	//
	// For some reason this is provided in the body
	// instead of a HTTP status code :|
	Code int `json:"code"`
}
