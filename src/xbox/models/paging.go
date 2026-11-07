package models

// PagingInfo represents the paging information returned by the Xbox API.
type PagingInfo struct {
	// ContinuationToken is the token used to fetch the next page of results.
	ContinuationToken string `json:"continuationToken"`
}
