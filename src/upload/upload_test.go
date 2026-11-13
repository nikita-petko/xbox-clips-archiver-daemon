package upload

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/dl"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/s3"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

func TestUploadGroupedMedia(t *testing.T) {
	oldCache, oldPrefix := *flags.CachePath, *flags.S3PathPrefix
	oldBucket, oldRegion := *flags.S3Bucket, *flags.S3Region
	oldAccess, oldSecret := *flags.S3AccessKey, *flags.S3SecretAccessKey
	oldUserMode, oldTitleMode := *flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode
	oldFileMode := *flags.UploadMediaFileNameMode
	oldVerify := *flags.ShouldVerifyUploadIntegrity
	oldAlertURLBase := *flags.AlertURLBase
	t.Cleanup(func() {
		*flags.CachePath, *flags.S3PathPrefix = oldCache, oldPrefix
		*flags.S3Bucket, *flags.S3Region = oldBucket, oldRegion
		*flags.S3AccessKey, *flags.S3SecretAccessKey = oldAccess, oldSecret
		*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = oldUserMode, oldTitleMode
		*flags.UploadMediaFileNameMode = oldFileMode
		*flags.ShouldVerifyUploadIntegrity = oldVerify
		*flags.AlertURLBase = oldAlertURLBase
	})
	*flags.ShouldVerifyUploadIntegrity = false
	*flags.AlertURLBase = ""
	*flags.CachePath, *flags.S3PathPrefix = t.TempDir(), "/archive/"
	*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = "id", "id"
	*flags.UploadMediaFileNameMode = "captured"
	cache.Setup()
	t.Cleanup(cache.Close)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "clip data")
	}))
	defer source.Close()
	captured := time.Date(2024, 1, 2, 0, 30, 0, 0, time.FixedZone("local", 3600))
	item := func(id string, cached bool) *dl.UserMediaCacheGuard {
		t.Helper()
		timestamp := captured
		switch id {
		case "missing":
			timestamp = captured.Add(time.Second)
		case "existing":
			timestamp = captured.Add(2 * time.Second)
		}
		guard, err := cache.CacheGuardedHttpGet("user", "42", id, source.URL+"/video file.mp4")
		if err != nil {
			t.Fatal(err)
		}
		return &dl.UserMediaCacheGuard{
			Media:      &models.Media{Id: id, TitleId: 42, CreatedAt: &timestamp, MediaUris: []*models.DownloadURI{{URI: source.URL}}},
			CachedFile: guard, WasCached: cached,
		}
	}
	newItem, missingItem, existingItem := item("new", false), item("missing", true), item("existing", true)
	noURI := &dl.UserMediaCacheGuard{Media: &models.Media{Id: "no-uri"}}

	var mu sync.Mutex
	requests := make(map[string]int)
	checksumMode := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.Method+" "+r.URL.Path]++
		mode := checksumMode
		mu.Unlock()
		switch {
		case r.Method == http.MethodHead && r.Header.Get("X-Amz-Checksum-Mode") == "ENABLED":
			switch mode {
			case "valid", "mismatch":
				content := "clip data"
				if mode == "mismatch" {
					content = "different"
				}
				sum := sha256.Sum256([]byte(content))
				w.Header().Set("X-Amz-Checksum-Sha256", base64.StdEncoding.EncodeToString(sum[:]))
			case "error":
				w.WriteHeader(http.StatusForbidden)
			}
		case r.Method == http.MethodHead && r.URL.Path == "/clips/archive/user/42/2024-01-01T23-30-02+0000.mp4":
		case r.Method == http.MethodHead && r.URL.Path == "/clips/archive/Game Title/2024-01-01T23-30-00+0000.mp4":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodHead && r.URL.Path == "/clips/archive/Player Name/Game Title/2024-01-01T23-30-00+0000.mp4":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodHead && r.URL.Path == "/clips/archive/user/42/2024-01-01T23-30-01+0000.mp4":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodHead && r.URL.Path == "/clips/archive/user/42/new.mp4":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodHead && strings.TrimSuffix(r.URL.Path, "/") == "/clips":
		case r.Method == http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "clip data" || r.Header.Get("Content-Type") != "video/mp4" {
				http.Error(w, "unexpected body", http.StatusBadRequest)
			}
		default:
			http.Error(w, "unexpected request", http.StatusForbidden)
		}
	}))
	defer server.Close()
	t.Setenv("AWS_ENDPOINT_URL_S3", server.URL)
	*flags.S3Bucket, *flags.S3Region = "clips", "us-east-1"
	*flags.S3AccessKey, *flags.S3SecretAccessKey = "access", "secret"
	s3.Setup()

	uploaded, err := UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{
		"user": {newItem, missingItem, existingItem, noURI},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploaded) != 2 {
		t.Fatalf("got %d uploaded items, want 2", len(uploaded))
	}
	for _, entry := range []*dl.UserMediaCacheGuard{newItem, missingItem} {
		want := fmt.Sprintf("s3://clips/archive/user/42/%s.mp4", entry.Media.CreatedAt.UTC().Format("2006-01-02T15-04-05-0700"))
		if uploaded[entry.Media] != want {
			t.Errorf("URI = %q, want %q", uploaded[entry.Media], want)
		}
	}
	mu.Lock()
	for request, want := range map[string]int{
		"HEAD /clips/archive/user/42/2024-01-01T23-30-00+0000.mp4": 0,
		"HEAD /clips/archive/user/42/2024-01-01T23-30-01+0000.mp4": 1,
		"HEAD /clips/archive/user/42/2024-01-01T23-30-02+0000.mp4": 1,
		"PUT /clips/archive/user/42/2024-01-01T23-30-00+0000.mp4":  1,
		"PUT /clips/archive/user/42/2024-01-01T23-30-01+0000.mp4":  1,
		"PUT /clips/archive/user/42/2024-01-01T23-30-02+0000.mp4":  0,
	} {
		if requests[request] != want {
			t.Errorf("%s: got %d requests, want %d", request, requests[request], want)
		}
	}
	mu.Unlock()

	t.Run("shared session rejects cross-user key collisions", func(t *testing.T) {
		*flags.UploaderGroupByUserMode = "none"
		defer func() { *flags.UploaderGroupByUserMode = "id" }()
		session := new(Session)
		first, err := session.UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{"first": {newItem}})
		if err != nil || len(first.Files) != 1 {
			t.Fatalf("first report = %+v, error = %v", first, err)
		}
		copy := *newItem
		media := *newItem.Media
		copy.Media = &media
		second, err := session.UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{"second": {&copy}})
		if err == nil || !strings.Contains(err.Error(), "occurs more than once") || len(second.Files) != 0 {
			t.Fatalf("second report = %+v, error = %v", second, err)
		}
	})

	t.Run("concurrent cross-user key collisions", func(t *testing.T) {
		*flags.UploaderGroupByUserMode = "none"
		defer func() { *flags.UploaderGroupByUserMode = "id" }()
		session := new(Session)
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, userID := range []string{"first", "second"} {
			copy := *newItem
			media := *newItem.Media
			copy.Media = &media
			go func() {
				<-start
				_, err := session.UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{userID: {&copy}})
				results <- err
			}()
		}
		close(start)
		var successes, collisions int
		for range 2 {
			err := <-results
			switch {
			case err == nil:
				successes++
			case strings.Contains(err.Error(), "occurs more than once"):
				collisions++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}
		if successes != 1 || collisions != 1 {
			t.Errorf("got %d successes and %d collisions", successes, collisions)
		}
	})

	t.Run("alert URL override", func(t *testing.T) {
		defer func() {
			*flags.AlertURLBase = ""
			*flags.ShouldVerifyUploadIntegrity = false
		}()
		*flags.AlertURLBase = "https://share.cdn.vmminfra.dev/"
		report, err := UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{"user": {newItem}})
		if err != nil || len(report.Files) != 1 {
			t.Fatalf("report = %+v, error = %v", report, err)
		}
		if report.Files[0].URL != "https://share.cdn.vmminfra.dev/archive/user/42/2024-01-01T23-30-00%2B0000.mp4" ||
			report.Files[0].URI != "s3://clips/archive/user/42/2024-01-01T23-30-00+0000.mp4" {
			t.Fatalf("unexpected report file: %+v", report.Files[0])
		}

		*flags.ShouldVerifyUploadIntegrity = true
		mu.Lock()
		checksumMode = "mismatch"
		mu.Unlock()
		report, err = UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{"user": {newItem}})
		if err == nil || !strings.Contains(err.Error(), "checksum mismatch") || len(report.Files) != 1 ||
			report.Files[0].URL != "https://share.cdn.vmminfra.dev/archive/user/42/2024-01-01T23-30-00%2B0000.mp4" {
			t.Fatalf("verification failure report = %+v, error = %v", report, err)
		}

		*flags.AlertURLBase = "http://insecure.example.com"
		mu.Lock()
		before := requests["PUT /clips/archive/user/42/2024-01-01T23-30-00+0000.mp4"]
		mu.Unlock()
		report, err = UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{"user": {newItem}})
		if err == nil || len(report.Files) != 0 {
			t.Fatalf("invalid base report = %+v, error = %v", report, err)
		}
		mu.Lock()
		after := requests["PUT /clips/archive/user/42/2024-01-01T23-30-00+0000.mp4"]
		mu.Unlock()
		if after != before {
			t.Error("upload occurred despite invalid alert URL base")
		}
	})

	t.Run("integrity and report", func(t *testing.T) {
		defer func() { *flags.ShouldVerifyUploadIntegrity = false }()
		for _, tc := range []struct {
			name, mode, wantError string
			verify, verified      bool
			wantHeads             int
		}{
			{"disabled", "", "", false, false, 0},
			{"valid", "valid", "", true, true, 1},
			{"mismatch", "mismatch", "checksum mismatch", true, false, 1},
			{"missing checksum", "", "no supported checksum", true, false, 1},
			{"head error", "error", "retrieve S3 checksum", true, false, 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				*flags.ShouldVerifyUploadIntegrity = tc.verify
				mu.Lock()
				checksumMode = tc.mode
				before := requests["HEAD /clips/archive/user/42/2024-01-01T23-30-00+0000.mp4"]
				mu.Unlock()
				report, err := UploadGroupedMediaReport(context.Background(), dl.UserGroupedMedia{"user": {newItem}})
				if tc.wantError == "" && err != nil || tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				if len(report.Files) != 1 || report.Files[0].Bytes != int64(len("clip data")) ||
					report.Files[0].Verified != tc.verified || report.VerificationEnabled != tc.verify ||
					report.Files[0].URL != "https://s3.us-east-1.amazonaws.com/clips/archive/user/42/2024-01-01T23-30-00%2B0000.mp4" ||
					report.Duration <= 0 {
					t.Fatalf("unexpected report: %+v", report)
				}
				mu.Lock()
				gotHeads := requests["HEAD /clips/archive/user/42/2024-01-01T23-30-00+0000.mp4"] - before
				mu.Unlock()
				if gotHeads != tc.wantHeads {
					t.Errorf("verification HEADs = %d, want %d", gotHeads, tc.wantHeads)
				}
			})
		}
	})

	t.Run("default layout", func(t *testing.T) {
		*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = "none", "name"
		copy := *newItem
		title := *newItem.Media
		title.TitleName = "Game Title"
		copy.Media, copy.WasCached = &title, true
		result, err := UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{"user": {&copy}})
		if err != nil || result[&title] != "s3://clips/archive/Game%20Title/2024-01-01T23-30-00+0000.mp4" {
			t.Fatalf("default layout result = %v, error %v", result, err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, method := range []string{"HEAD", "PUT"} {
			request := method + " /clips/archive/Game Title/2024-01-01T23-30-00+0000.mp4"
			if requests[request] != 1 {
				t.Errorf("%s: got %d requests, want 1", request, requests[request])
			}
		}
	})
	t.Run("gamertag layout", func(t *testing.T) {
		*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = "gamertag", "name"
		copy := *newItem
		title := *newItem.Media
		title.TitleName = "Game Title"
		copy.Media, copy.WasCached, copy.Gamertag = &title, true, "Player Name"
		result, err := UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{"user": {&copy}})
		if err != nil || result[&title] != "s3://clips/archive/Player%20Name/Game%20Title/2024-01-01T23-30-00+0000.mp4" {
			t.Fatalf("gamertag layout result = %v, error %v", result, err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, method := range []string{"HEAD", "PUT"} {
			request := method + " /clips/archive/Player Name/Game Title/2024-01-01T23-30-00+0000.mp4"
			if requests[request] != 1 {
				t.Errorf("%s: got %d requests, want 1", request, requests[request])
			}
		}
	})
	*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = "id", "id"

	t.Run("id filename", func(t *testing.T) {
		*flags.UploadMediaFileNameMode = "id"
		defer func() { *flags.UploadMediaFileNameMode = "captured" }()
		copy := *newItem
		copy.WasCached = true
		result, err := UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{"user": {&copy}})
		if err != nil || result[copy.Media] != "s3://clips/archive/user/42/new.mp4" {
			t.Fatalf("id layout result = %v, error %v", result, err)
		}
		mu.Lock()
		defer mu.Unlock()
		for _, method := range []string{"HEAD", "PUT"} {
			request := method + " /clips/archive/user/42/new.mp4"
			if requests[request] != 1 {
				t.Errorf("%s: got %d requests, want 1", request, requests[request])
			}
		}
	})

	t.Run("legacy capture timestamp", func(t *testing.T) {
		for _, tc := range []struct {
			name, input, want string
		}{
			{"UTC", "2024-06-07T08:09:10Z", "2024-06-07T08-09-10+0000.mp4"},
			{"fractional", "2024-06-07T08:09:10.987654Z", "2024-06-07T08-09-10+0000.mp4"},
			{"offset rollover", "2024-01-02T00:30:00+01:00", "2024-01-01T23-30-00+0000.mp4"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				date, err := time.Parse(time.RFC3339Nano, tc.input)
				if err != nil {
					t.Fatal(err)
				}
				item := *newItem
				media := *newItem.Media
				media.CreatedAt = &date
				item.Media = &media
				key, err := objectKey("user", &item)
				if want := "archive/user/42/" + tc.want; err != nil || key != want {
					t.Errorf("key = %q, want %q; error = %v", key, want, err)
				}
			})
		}
	})

	t.Run("filename errors and escaping", func(t *testing.T) {
		copy := *newItem
		title := *newItem.Media
		copy.Media = &title
		for _, date := range []*time.Time{nil, new(time.Time)} {
			title.CreatedAt = date
			if _, err := objectKey("user", &copy); err == nil || !strings.Contains(err.Error(), "capture date") {
				t.Errorf("date %v: got %v", date, err)
			}
		}
		*flags.UploadMediaFileNameMode = "id"
		title.Id = "../a%b"
		key, err := objectKey("user", &copy)
		if err != nil || key != "archive/user/42/..%2Fa%25b.mp4" {
			t.Errorf("escaped filename = %q, error %v", key, err)
		}
		guard, err := cache.CacheGuardedHttpGet("user", "42", "noext", source.URL+"/noext")
		if err != nil {
			t.Fatal(err)
		}
		copy.CachedFile = guard
		if _, err := objectKey("user", &copy); err == nil || !strings.Contains(err.Error(), "extension") {
			t.Errorf("missing extension: %v", err)
		}
		*flags.UploadMediaFileNameMode = "invalid"
		if _, err := objectKey("user", &copy); err == nil || !strings.Contains(err.Error(), "file name mode") {
			t.Errorf("invalid mode: %v", err)
		}
		*flags.UploadMediaFileNameMode = "captured"
	})

	t.Run("errors", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := CollectMedia(ctx, "user", []*dl.UserMediaCacheGuard{newItem}); !errors.Is(err, context.Canceled) {
			t.Errorf("canceled collection: %v", err)
		}
		if _, err := UploadGroupedMedia(ctx, dl.UserGroupedMedia{"user": nil}); !errors.Is(err, context.Canceled) {
			t.Errorf("canceled empty collection: %v", err)
		}
		if _, err := CollectMedia(context.Background(), "user", []*dl.UserMediaCacheGuard{nil}); err == nil {
			t.Error("expected nil media error")
		}
		if _, err := CollectMedia(context.Background(), "user", []*dl.UserMediaCacheGuard{{Media: newItem.Media}}); err == nil {
			t.Error("expected missing cached file error")
		}
		if _, err := UploadMedia(context.Background(), Media{}); err == nil {
			t.Error("expected invalid upload error")
		}
		if _, err := CollectMedia(context.Background(), "user", []*dl.UserMediaCacheGuard{{
			Media: &models.Media{Id: "forbidden", TitleId: 42, CreatedAt: &captured}, CachedFile: existingItem.CachedFile, WasCached: true,
		}}); err == nil {
			t.Error("expected S3 HEAD error")
		}
		duplicate := *newItem.Media
		result, err := UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{
			"user": {newItem, {Media: &duplicate, CachedFile: newItem.CachedFile}},
		})
		if err == nil || len(result) != 1 {
			t.Errorf("duplicate key: got %d completed uploads, error %v", len(result), err)
		}
		if err := os.Remove(newItem.CachedFile.GetFilePath()); err != nil {
			t.Fatal(err)
		}
		if _, err := UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{"user": {newItem}}); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected missing file error, got %v", err)
		}
		result, err = UploadGroupedMedia(context.Background(), dl.UserGroupedMedia{"user": {missingItem, newItem}})
		if !errors.Is(err, os.ErrNotExist) || len(result) != 1 || result[missingItem.Media] == "" {
			t.Errorf("partial upload: got %v, error %v", result, err)
		}
	})

	t.Run("escaped key", func(t *testing.T) {
		media := &dl.UserMediaCacheGuard{
			Media: &models.Media{Id: "../a%b", TitleId: 42, CreatedAt: &captured}, CachedFile: existingItem.CachedFile,
		}
		key, err := objectKey("a/b", media)
		if err != nil || key != "archive/a%2Fb/42/2024-01-01T23-30-00+0000.mp4" {
			t.Fatalf("key = %q, error %v", key, err)
		}
		uri, err := s3.URI(key)
		if err != nil || uri != "s3://clips/archive/a%252Fb/42/2024-01-01T23-30-00+0000.mp4" {
			t.Errorf("URI = %q, error %v", uri, err)
		}
	})

	t.Run("grouping modes", func(t *testing.T) {
		media := &dl.UserMediaCacheGuard{
			Gamertag:   "Gamer/Tag",
			Media:      &models.Media{Id: "clip", TitleId: 42, TitleName: "Game % Deluxe", CreatedAt: &captured},
			CachedFile: existingItem.CachedFile,
		}
		for _, userMode := range []struct{ mode, part string }{
			{"none", ""}, {"id", "user/"}, {"gamertag", "Gamer%2FTag/"},
		} {
			for _, titleMode := range []struct{ mode, part string }{
				{"none", ""}, {"id", "42/"}, {"name", "Game %25 Deluxe/"},
			} {
				t.Run(userMode.mode+"_"+titleMode.mode, func(t *testing.T) {
					*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = userMode.mode, titleMode.mode
					key, err := objectKey("user", media)
					want := "archive/" + userMode.part + titleMode.part + "2024-01-01T23-30-00+0000.mp4"
					if err != nil || key != want {
						t.Fatalf("key = %q, want %q; error = %v", key, want, err)
					}
				})
			}
		}
		for _, tc := range []struct {
			userMode, titleMode, userID, gamertag, titleName string
		}{
			{"invalid", "name", "user", "tag", "title"},
			{"none", "invalid", "user", "tag", "title"},
			{"id", "none", "", "tag", "title"},
			{"gamertag", "none", "user", "", "title"},
			{"none", "name", "user", "tag", ""},
		} {
			*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode = tc.userMode, tc.titleMode
			copy := *media
			copy.Gamertag = tc.gamertag
			title := *media.Media
			title.TitleName = tc.titleName
			copy.Media = &title
			if _, err := objectKey(tc.userID, &copy); err == nil {
				t.Errorf("expected error for %+v", tc)
			}
		}
	})
}
