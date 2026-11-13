package models

// ProfileUserSetting represents a setting associated with a profile user.
type ProfileUserSetting struct {
	// Id is the unique identifier of the profile user setting.
	Id string `json:"id"`

	// Value is the value of the profile user setting.
	Value string `json:"value"`
}

// ProfileUser represents a user profile in the Xbox system.
type ProfileUser struct {
	// Id is the unique identifier of the profile user.
	Id string `json:"id"`

	// Settings contains the settings associated with the profile user.
	Settings []*ProfileUserSetting `json:"settings"`
}

// ProfileUserContent represents the content of a profile user response returned by the Xbox API.
type ProfileUserContent struct {
	// ProfileUsers contains the profile user information.
	ProfileUsers []*ProfileUser `json:"profileUsers"`
}

// ProfileUserResponse represents the response structure returned by the Xbox API for profile users.
type ProfileUserResponse struct {
	// Content contains the profile user content.
	Content *ProfileUserContent `json:"content"`

	// Code is the status code returned by the API.
	//
	// For some reason this is provided in the body
	// instead of a HTTP status code :|
	Code int `json:"code"`
}
