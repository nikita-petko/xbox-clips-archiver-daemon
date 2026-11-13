package dvr

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

const screenshotsEndpoint = "/v2/dvr/screenshots"
const gameclipsEndpoint = "/v2/dvr/gameclips"

func parseCaptureTime(value string) (*time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse("2006-01-02 15:04:05Z", value)
		if err != nil {
			return nil, fmt.Errorf("invalid capture timestamp %q: %w", value, err)
		}
	}
	return &parsed, nil
}

// getDVRScreenshots retrieves a list of DVR screenshots for the authenticated user using the provided API key.
func getDVRScreenshots(apiKey string) ([]*models.Screenshot, error) {
	continuationToken := ""
	screenshots := make([]*models.Screenshot, 0)

	for {
		data, newContinuationToken, err := xbl.MakeApiCall(fmt.Sprintf("%s?continuationToken=%s", screenshotsEndpoint, continuationToken), apiKey, true)
		if err != nil {
			return nil, err
		}

		continuationToken = newContinuationToken
		if len(data) > 0 {
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
		data, newContinuationToken, err := xbl.MakeApiCall(fmt.Sprintf("%s?continuationToken=%s", gameclipsEndpoint, continuationToken), apiKey, true)
		if err != nil {
			return nil, err
		}

		continuationToken = newContinuationToken
		if len(data) > 0 {
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

// GetDVRMedia retrieves a combined list of DVR media (screenshots and game clips) for the authenticated user using the provided API key.
func GetDVRMedia(apiKey string) ([]*models.Media, error) {
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
			parsedTime, err := parseCaptureTime(s.DateTaken)
			if err == nil {
				taken = parsedTime
			} else {
				glog.Warningf("Invalid capture timestamp for screenshot %q: %v", s.ScreenshotId, err)
			}
		}

		media = append(media, &models.Media{
			Category:  models.MediaCategoryScreenshot,
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
			parsedTime, err := parseCaptureTime(g.DateRecorded)
			if err == nil {
				recorded = parsedTime
			} else {
				glog.Warningf("Invalid capture timestamp for game clip %q: %v", g.GameClipId, err)
			}
		}

		media = append(media, &models.Media{
			Category:  models.MediaCategoryGameClip,
			Id:        g.GameClipId,
			TitleId:   g.TitleId,
			TitleName: g.TitleName,
			CreatedAt: recorded,
			MediaUris: g.GameClipUris,
		})
	}

	return media, nil
}
