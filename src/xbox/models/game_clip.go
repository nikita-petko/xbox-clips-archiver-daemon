package models

// GameClip represents the structure of a screenshot returned by the Xbox API
// filled in with only required information
type GameClip struct {
	// GameClipId is the unique identifier of the screenshot.
	//
	// Used within the cache to ensure that
	// each screenshot is uniquely identified and
	// isn't duplicated within the cache.
	GameClipId string `json:"screenshotId"`

	// DateTaken is the timestamp when the screenshot was taken.
	//
	// It serves as the name of the screenshot on S3.
	DateTaken string `json:"dateTaken"`

	// TitleId is the identifier of the game title associated with the screenshot.
	//
	// Only relevant if GroupByTitleId is true, otherwise TitleName is used.
	TitleId int `json:"titleId"`

	// TitleName is the name of the game title associated with the screenshot.
	//
	// Only relevant if GroupByTitleId is false, otherwise TitleId is used.
	TitleName string `json:"titleName"`

	// GameClipUris is a list of download URIs for the screenshot.
	GameClipUris []*DownloadURI `json:"screenshotUris"`
}

// GameClipContent represents the actual content of a screenshot response returned by the Xbox API.
type GameClipContent struct {
	// GameClips is the list of screenshots returned by the API.
	GameClips []*GameClip `json:"gameClips"`

	// PagingInfo contains the paging information for the screenshot results.
	PagingInfo *PagingInfo `json:"pagingInfo"`
}

// GameClipResponse represents the response structure returned by the Xbox API for screenshots.
type GameClipResponse struct {
	// Content contains the actual screenshot content.
	Content *GameClipContent `json:"content"`

	// Code is the status code returned by the API.
	//
	// For some reason this is provided in the body
	// instead of a HTTP status code :|
	Code int `json:"code"`
}
