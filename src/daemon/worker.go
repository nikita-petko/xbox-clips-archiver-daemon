package daemon

import (
	"context"
	"fmt"
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
	users := users.GetUsersMap()
	if len(users) == 0 {
		return fmt.Errorf("no users found")
	}

	groupedMedia, err := dl.DownloadAllUserMedias(users)
	if err != nil {
		return fmt.Errorf("failed to download user medias: %w", err)
	}

	ctx := context.Background()
	report, err := upload.UploadGroupedMediaReport(ctx, groupedMedia)
	sendUploadAlert(ctx, report, err)
	if err != nil {
		return fmt.Errorf("failed to upload user media: %w", err)
	}
	glog.Infof("Uploaded %d media items", len(report.Files))

	return nil
}

func sendUploadAlert(ctx context.Context, report upload.Report, err error) {
	if len(report.Files) > 0 || err != nil {
		for _, description := range uploadAlertDescriptions(report, err) {
			alerting.Alert(ctx, "Xbox media upload", description)
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
