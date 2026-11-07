package xbox

import (
	"encoding/json"
	"fmt"
	"io"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbox/models"
)

// GetDVRScreenshots retrieves a list of DVR screenshots for the authenticated user using the provided API key.
func GetDVRScreenshots(apiKey string) ([]*models.Screenshot, error) {
	continuationToken := ""
	screenshots := make([]*models.Screenshot, 0)

	for {
		httpResponse, newContinuationToken, err := makeApiCall(fmt.Sprintf("%s?continuationToken=%s", SCREENSHOTS_ENDPOINT, continuationToken), apiKey)
		if err != nil {
			return nil, err
		}

		continuationToken = newContinuationToken
		if httpResponse != nil {
			data, err := io.ReadAll(httpResponse.Body)
			if err != nil {
				return nil, err
			}

			var response models.ScreenshotResponse
			err = json.Unmarshal(data, &response)
			if err != nil {
				return nil, err
			}
			screenshots = append(screenshots, response.Content.Screenshots...)
		}

		if continuationToken == "" {
			break
		}
	}

	return screenshots, nil
}

// GetDVRGameClips retrieves a list of DVR game clips for the authenticated user using the provided API key.
func GetDVRGameClips(apiKey string) ([]*models.GameClip, error) {
	continuationToken := ""
	gameClips := make([]*models.GameClip, 0)

	for {
		httpResponse, newContinuationToken, err := makeApiCall(fmt.Sprintf("%s?continuationToken=%s", GAMECLIPS_ENDPOINT, continuationToken), apiKey)
		if err != nil {
			return nil, err
		}

		continuationToken = newContinuationToken
		if httpResponse != nil {
			data, err := io.ReadAll(httpResponse.Body)
			if err != nil {
				return nil, err
			}

			var response models.GameClipResponse
			err = json.Unmarshal(data, &response)
			if err != nil {
				return nil, err
			}
			gameClips = append(gameClips, response.Content.GameClips...)
		}

		if continuationToken == "" {
			break
		}
	}

	return gameClips, nil
}
