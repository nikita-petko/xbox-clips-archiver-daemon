package users

import (
	"encoding/json"
	"errors"
	"io"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

const accountEndpoint = "/v2/account"
const userProfileGamertagSettingId = "Gamertag" // The setting ID for the user's Gamertag in their profile.

// User represents an Xbox user with a user ID and gamertag.
type User struct {
	UserID   string
	Gamertag string
}

// ApiKeyToUserMap maps API keys to their corresponding Xbox users.
type ApiKeyToUserMap map[string]User

func getUserProfile(apiKey string) (string, string, error) {
	if profile, ok := cache.GetUserProfile(apiKey); ok {
		return profile.UserID, profile.Gamertag, nil
	}

	httpResponse, _, err := xbl.MakeApiCall(accountEndpoint, apiKey, false)
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
		if setting != nil && setting.Id == userProfileGamertagSettingId {
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

// GetUsersByApiKeys retrieves a map of API keys to their corresponding Xbox users
func GetUsersByApiKeys(apiKeys []string) (ApiKeyToUserMap, error) {
	result := make(ApiKeyToUserMap)

	for _, apiKey := range apiKeys {
		userID, gamertag, err := getUserProfile(apiKey)
		if err != nil {
			return nil, err
		}
		result[apiKey] = User{UserID: userID, Gamertag: gamertag}
	}
	return result, nil
}
