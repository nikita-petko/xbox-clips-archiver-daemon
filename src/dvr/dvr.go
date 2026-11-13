package dvr

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

const screenshotsEndpoint = "/v2/dvr/screenshots"
const gameclipsEndpoint = "/v2/dvr/gameclips"

// getDVRScreenshots retrieves a list of DVR screenshots for the authenticated user using the provided API key.
func getDVRScreenshots(apiKey string) ([]*models.Screenshot, error) {
	continuationToken := ""
	screenshots := make([]*models.Screenshot, 0)

	for {
		httpResponse, newContinuationToken, err := xbl.MakeApiCall(fmt.Sprintf("%s?continuationToken=%s", screenshotsEndpoint, continuationToken), apiKey, true)
		if err != nil {
			return nil, err
		}

		continuationToken = newContinuationToken
		if httpResponse != nil {
			defer httpResponse.Body.Close()

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

// getDVRGameClips retrieves a list of DVR game clips for the authenticated user using the provided API key.
func getDVRGameClips(apiKey string) ([]*models.GameClip, error) {
	continuationToken := ""
	gameClips := make([]*models.GameClip, 0)

	for {
		httpResponse, newContinuationToken, err := xbl.MakeApiCall(fmt.Sprintf("%s?continuationToken=%s", gameclipsEndpoint, continuationToken), apiKey, true)
		if err != nil {
			return nil, err
		}

		continuationToken = newContinuationToken
		if httpResponse != nil {
			defer httpResponse.Body.Close()

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

// getDVRMedia retrieves a combined list of DVR media (screenshots and game clips) for the authenticated user using the provided API key.
func getDVRMedia(apiKey string) ([]*models.Media, error) {
	screenshots, err := getDVRScreenshots(apiKey)
	if err != nil {
		return nil, err
	}

	gameClips, err := getDVRGameClips(apiKey)
	if err != nil {
		return nil, err
	}

	media := make([]*models.Media, 0, len(screenshots)+len(gameClips))
	for _, s := range screenshots {
		taken := &time.Time{}
		if s.DateTaken != "" {
			parsedTime, err := time.Parse(time.RFC3339, s.DateTaken)
			if err == nil {
				taken = &parsedTime
			}
		}

		media = append(media, &models.Media{
			Id:        s.ScreenshotId,
			TitleId:   s.TitleId,
			TitleName: s.TitleName,
			CreatedAt: taken,
			MediaUris: s.ScreenshotUris,
		})
	}
	for _, g := range gameClips {
		recorded := &time.Time{}
		if g.DateRecorded != "" {
			parsedTime, err := time.Parse(time.RFC3339, g.DateRecorded)
			if err == nil {
				recorded = &parsedTime
			}
		}

		media = append(media, &models.Media{
			Id:        g.GameClipId,
			TitleId:   g.TitleId,
			TitleName: g.TitleName,
			CreatedAt: recorded,
			MediaUris: g.GameClipUris,
		})
	}

	return media, nil
}

// GetGroupedDVRMedia retrieves DVR media grouped by game ID for the authenticated user using the provided API key.
//
// Also returns a game ID to game title map.
func GetGroupedDVRMedia(apiKey string) (map[string][]*models.Media, map[string]string, error) {
	media, err := getDVRMedia(apiKey)
	if err != nil {
		return nil, nil, err
	}

	groupedMedia := make(map[string][]*models.Media)
	gameTitles := make(map[string]string)
	for _, m := range media {
		gameID := fmt.Sprintf("%d", m.TitleId)
		groupedMedia[gameID] = append(groupedMedia[gameID], m)
		gameTitles[gameID] = m.TitleName
	}

	return groupedMedia, gameTitles, nil
}
