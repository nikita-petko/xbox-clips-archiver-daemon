package daemon

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/dl"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/upload"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/users"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

// DoWork is the function that is intended to perform the main work of the daemon.
// It is exported to allow single run work.
func DoWork() error {
	ctx := context.Background()
	session := new(upload.Session)
	return doWork(users.GetUsersMap(), func(apiKeys []string, user users.User) error {
		return doUserWork(ctx, session, apiKeys, user)
	})
}

func doWork(usersMap users.ApiKeyToUserMap, work func([]string, users.User) error) error {
	if len(usersMap) == 0 {
		return fmt.Errorf("no users found")
	}

	// Multiple API keys can resolve to the same user; run that user's pipeline once.
	keys := make([]string, 0, len(usersMap))
	for key := range usersMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	type userWork struct {
		user    users.User
		apiKeys []string
	}
	grouped := make(map[string]*userWork)
	for _, key := range keys {
		user := usersMap[key]
		if grouped[user.UserID] == nil {
			grouped[user.UserID] = &userWork{user: user}
		}
		grouped[user.UserID].apiKeys = append(grouped[user.UserID].apiKeys, key)
	}
	results := make(chan error, len(grouped))
	for _, job := range grouped {
		go func() {
			if err := work(job.apiKeys, job.user); err != nil {
				results <- fmt.Errorf("user %q: %w", job.user.UserID, err)
			} else {
				results <- nil
			}
		}()
	}
	var errs []error
	for range len(grouped) {
		if err := <-results; err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func doUserWork(ctx context.Context, session *upload.Session, apiKeys []string, user users.User) error {
	var media []*dl.UserMediaCacheGuard
	var err error
	for _, apiKey := range apiKeys {
		media, err = dl.DownloadUserMedia(apiKey, user)
		if err == nil {
			break
		}
		glog.Errorf("failed to download media for user %q: %v", user.UserID, err)
	}
	if err != nil {
		return nil
	}
	report, err := session.UploadGroupedMediaReport(ctx, dl.UserGroupedMedia{user.UserID: media})
	sendUploadAlert(ctx, user, report, err)
	if err != nil {
		return fmt.Errorf("failed to upload user media: %w", err)
	}
	glog.Infof("Uploaded %d media items for user %q", len(report.Files), user.UserID)

	return nil
}

func sendUploadAlert(ctx context.Context, user users.User, report upload.Report, err error) {
	if len(report.Files) > 0 || err != nil {
		for _, description := range uploadAlertDescriptions(report, err) {
			alerting.Alert(ctx, fmt.Sprintf("Xbox media upload: %s (%s)", user.Gamertag, user.UserID), description)
		}
	}
}

func uploadAlertDescriptions(report upload.Report, uploadErr error) []string {
	var screenshots, clips, other int
	var bytes int64
	for _, file := range report.Files {
		switch file.Media.Category {
		case models.MediaCategoryScreenshot:
			screenshots++
		case models.MediaCategoryGameClip:
			clips++
		default:
			other++
		}
		bytes += file.Bytes
	}
	verification := "disabled"
	if report.VerificationEnabled {
		verification = "passed"
		if len(report.Files) == 0 {
			verification = "not performed"
		}
		for _, file := range report.Files {
			if !file.Verified {
				verification = "failed"
				break
			}
		}
	}
	summary := fmt.Sprintf("Uploaded: %d files (screenshots: %d, game clips: %d, other: %d)\nTotal bytes: %d\nDuration: %s\nIntegrity verification: %s",
		len(report.Files), screenshots, clips, other, bytes, report.Duration, verification)
	if uploadErr != nil {
		message := []rune(uploadErr.Error())
		if len(message) > 500 {
			message = append(message[:500], []rune("...")...)
		}
		summary += "\nUpload failed: " + string(message)
	}

	descriptions := []string{summary}
	for _, file := range report.Files {
		line := "\n" + file.URL
		last := len(descriptions) - 1
		if len(descriptions[last])+len(line) > 3500 {
			descriptions = append(descriptions, strings.TrimPrefix(line, "\n"))
		} else {
			descriptions[last] += line
		}
	}
	return descriptions
}
