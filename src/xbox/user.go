package xbox

import (
	"encoding/json"
	"errors"
	"io"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbox/models"
)

// GetUserProfile retrieves the Xbox user ID and gamertag associated with the provided API key.
func GetUserProfile(apiKey string) (string, string, error) {
	if profile, ok := cache.GetUserProfile(apiKey); ok {
		return profile.UserID, profile.Gamertag, nil
	}

	httpResponse, _, err := makeApiCall(ACCOUNT_ENDPOINT, apiKey, false)
	if err != nil {
		return "", "", err
	}

	defer httpResponse.Body.Close()
	data, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return "", "", err
	}

	var jsonResponse models.ProfileUserResponse
	err = json.Unmarshal(data, &jsonResponse)
	if err != nil {
		return "", "", err
	}

	if jsonResponse.Content == nil || len(jsonResponse.Content.ProfileUsers) == 0 || jsonResponse.Content.ProfileUsers[0] == nil {
		return "", "", errors.New("profile response did not contain a user")
	}

	user := jsonResponse.Content.ProfileUsers[0]
	if user.Id == "" {
		return "", "", errors.New("profile response did not contain a user ID")
	}

	gamertag := ""
	for _, setting := range user.Settings {
		if setting != nil && setting.Id == USERPROFILE_GAMERTAG_SETTING_ID {
			gamertag = setting.Value
			break
		}
	}
	if gamertag == "" {
		return "", "", errors.New("profile response did not contain a gamertag")
	}

	profile := cache.UserProfile{UserID: user.Id, Gamertag: gamertag}
	if err := cache.SetUserProfile(apiKey, profile); err != nil {
		return "", "", err
	}

	return profile.UserID, profile.Gamertag, nil
}
