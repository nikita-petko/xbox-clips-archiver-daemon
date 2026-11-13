package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/dl"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/s3"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

// Media identifies a cached file selected for upload and its S3 object key.
type Media struct {
	Media *models.Media
	File  *cache.CacheGuard
	Key   string
}

func objectKey(userID string, media *dl.UserMediaCacheGuard) (string, error) {
	if err := flags.ValidateUploaderGrouping(); err != nil {
		return "", err
	}
	if err := flags.ValidateUploadMediaFileNameMode(); err != nil {
		return "", err
	}
	if media == nil || media.Media == nil {
		return "", errors.New("media is required")
	}
	if media.Media.Id == "" {
		return "", errors.New("media ID is required")
	}
	if media.CachedFile == nil || media.CachedFile.GetFilePath() == "" {
		return "", errors.New("cached file is required")
	}
	parts := make([]string, 0, 4)
	switch *flags.UploaderGroupByUserMode {
	case "id":
		if userID == "" {
			return "", errors.New("user ID is required for user grouping")
		}
		parts = append(parts, userID)
	case "gamertag":
		if media.Gamertag == "" {
			return "", errors.New("gamertag is required for user grouping")
		}
		parts = append(parts, media.Gamertag)
	}
	switch *flags.UploaderGroupByTitleMode {
	case "id":
		parts = append(parts, strconv.Itoa(media.Media.TitleId))
	case "name":
		if media.Media.TitleName == "" {
			return "", errors.New("title name is required for title grouping")
		}
		parts = append(parts, media.Media.TitleName)
	}
	extension := strings.ToLower(filepath.Ext(media.CachedFile.GetFilePath()))
	if extension == "" || extension == "." {
		return "", errors.New("cached file extension is required")
	}
	filename := media.Media.Id
	if *flags.UploadMediaFileNameMode == "captured" {
		if media.Media.CreatedAt == nil || media.Media.CreatedAt.IsZero() {
			return "", errors.New("capture date is required for captured file naming")
		}
		filename = media.Media.CreatedAt.UTC().Format("2006-01-02T15-04-05-0700")
	}
	parts = append(parts, filename+extension)
	// Escaping each component keeps IDs containing separators from sharing keys.
	for i, part := range parts {
		if part == "." || part == ".." {
			parts[i] = strings.Repeat("%2E", len(part))
		} else {
			parts[i] = strings.NewReplacer("%", "%25", "/", "%2F").Replace(part)
		}
	}
	if prefix := strings.Trim(*flags.S3PathPrefix, "/"); prefix != "" {
		parts = append([]string{prefix}, parts...)
	}
	return strings.Join(parts, "/"), nil
}

// CollectMedia selects files for upload. Newly downloaded files are selected
// without a remote lookup; cached files are selected only if absent from S3.
// Media without download URIs is skipped. Lookup errors abort the collection.
// Keys use the configured prefix and grouping modes followed by the configured
// media filename.
func CollectMedia(ctx context.Context, userID string, mediaList []*dl.UserMediaCacheGuard) ([]Media, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	selected := make([]Media, 0, len(mediaList))
	for _, item := range mediaList {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if item != nil && item.Media != nil && item.CachedFile == nil && len(item.Media.MediaUris) == 0 {
			continue
		}
		key, err := objectKey(userID, item)
		if err != nil {
			return nil, fmt.Errorf("select media for user %q: %w", userID, err)
		}
		if item.WasCached {
			exists, err := s3.Exists(ctx, key)
			if err != nil {
				return nil, fmt.Errorf("check S3 object %q: %w", key, err)
			}
			if exists {
				continue
			}
		}
		selected = append(selected, Media{Media: item.Media, File: item.CachedFile, Key: key})
	}
	return selected, nil
}

// UploadMedia uploads a selected file and returns its S3 URI. If verification
// fails after the upload, it returns the URI along with the error.
func UploadMedia(ctx context.Context, media Media) (string, error) {
	if media.Media == nil || media.File == nil || media.Key == "" {
		return "", errors.New("media, cached file, and S3 key are required")
	}
	uri, err := s3.URI(media.Key)
	if err != nil {
		return "", fmt.Errorf("construct S3 URI for %q: %w", media.Key, err)
	}

	glog.Infof("Uploading media %q to S3 key %q", media.Media.Id, media.Key)

	if err := s3.Upload(ctx, media.Key, media.File); err != nil {
		return "", fmt.Errorf("upload S3 object %q: %w", media.Key, err)
	}
	if *flags.ShouldVerifyUploadIntegrity {
		valid, err := s3.ValidateIntegrity(ctx, media.Key, media.File)
		if err != nil {
			return uri, fmt.Errorf("verify S3 object %q: %w", media.Key, err)
		}
		if !valid {
			return uri, fmt.Errorf("verify S3 object %q: checksum mismatch", media.Key)
		}
	}
	return uri, nil
}

// UploadedMedia describes a completed upload, including one whose verification failed.
type UploadedMedia struct {
	Media    *models.Media
	URI      string
	URL      string
	Bytes    int64
	Verified bool
}

// Report describes the uploads completed during an upload pass.
type Report struct {
	Files               []UploadedMedia
	Duration            time.Duration
	VerificationEnabled bool
}

// Session tracks object keys across concurrent upload reports in one pass.
type Session struct {
	mu       sync.Mutex
	uploaded map[*models.Media]struct{}
	keys     map[string]struct{}
}

func (s *Session) reserve(media Media) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uploaded == nil {
		s.uploaded = make(map[*models.Media]struct{})
		s.keys = make(map[string]struct{})
	}
	if _, exists := s.uploaded[media.Media]; exists {
		return fmt.Errorf("media %q occurs more than once", media.Media.Id)
	}
	if _, exists := s.keys[media.Key]; exists {
		return fmt.Errorf("S3 object key %q occurs more than once", media.Key)
	}
	s.uploaded[media.Media] = struct{}{}
	s.keys[media.Key] = struct{}{}
	return nil
}

func (s *Session) release(media Media) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.uploaded, media.Media)
	delete(s.keys, media.Key)
}

func alertURL(ctx context.Context, key string) (string, error) {
	base, err := flags.ParseAlertURLBase()
	if err != nil {
		return "", err
	}
	if base == nil {
		return s3.HTTPSURL(ctx, key)
	}
	objectURL, err := s3.AppendObjectKey(base, key)
	if err != nil {
		return "", err
	}
	return objectURL.String(), nil
}

// UploadGroupedMedia selects and uploads each user's media. The returned map
// contains only newly uploaded media, keyed by the original media pointers.
// On error, the map includes uploads completed before or during the failure.
func UploadGroupedMedia(ctx context.Context, grouped dl.UserGroupedMedia) (map[*models.Media]string, error) {
	report, err := UploadGroupedMediaReport(ctx, grouped)
	uploaded := make(map[*models.Media]string)
	for _, file := range report.Files {
		uploaded[file.Media] = file.URI
	}
	return uploaded, err
}

// UploadGroupedMediaReport uploads media and reports all completed uploads,
// even if a subsequent upload or integrity check fails.
func UploadGroupedMediaReport(ctx context.Context, grouped dl.UserGroupedMedia) (report Report, err error) {
	return new(Session).UploadGroupedMediaReport(ctx, grouped)
}

// UploadGroupedMediaReport uploads media using this session's shared duplicate checks.
func (s *Session) UploadGroupedMediaReport(ctx context.Context, grouped dl.UserGroupedMedia) (report Report, err error) {
	started := time.Now()
	defer func() { report.Duration = time.Since(started) }()
	report.VerificationEnabled = *flags.ShouldVerifyUploadIntegrity
	if err := ctx.Err(); err != nil {
		return report, err
	}
	for userID, items := range grouped {
		selected, err := CollectMedia(ctx, userID, items)
		if err != nil {
			return report, err
		}
		for _, media := range selected {
			if err := s.reserve(media); err != nil {
				return report, err
			}
			info, err := os.Stat(media.File.GetFilePath())
			if err != nil {
				s.release(media)
				return report, fmt.Errorf("stat cached file for S3 object %q: %w", media.Key, err)
			}
			url, err := alertURL(ctx, media.Key)
			if err != nil {
				s.release(media)
				return report, fmt.Errorf("construct HTTPS URL for S3 object %q: %w", media.Key, err)
			}
			uri, err := UploadMedia(ctx, media)
			if uri != "" {
				report.Files = append(report.Files, UploadedMedia{
					Media: media.Media, URI: uri, URL: url, Bytes: info.Size(),
					Verified: report.VerificationEnabled && err == nil,
				})
			} else {
				s.release(media)
			}
			if err != nil {
				return report, err
			}
		}
	}
	return report, nil
}
