package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/upload"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/users"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

func TestUploadAlertDescriptions(t *testing.T) {
	report := upload.Report{
		Duration: time.Second, VerificationEnabled: true,
		Files: []upload.UploadedMedia{
			{Media: &models.Media{Category: models.MediaCategoryScreenshot}, URL: "https://example.com/image", Bytes: 5, Verified: true},
			{Media: &models.Media{Category: models.MediaCategoryGameClip}, URL: "https://example.com/video", Bytes: 10, Verified: false},
			{Media: &models.Media{}, URL: "https://example.com/other", Bytes: 3, Verified: true},
		},
	}
	got := uploadAlertDescriptions(report, errors.New("checksum mismatch"))
	for _, want := range []string{
		"3 files (screenshots: 1, game clips: 1, other: 1)", "Total bytes: 18", "Duration: 1s",
		"Integrity verification: failed", "Upload failed: checksum mismatch",
		"https://example.com/image", "https://example.com/video", "https://example.com/other",
	} {
		if !strings.Contains(strings.Join(got, "\n"), want) {
			t.Errorf("alert missing %q: %v", want, got)
		}
	}
	report.VerificationEnabled = false
	if got := uploadAlertDescriptions(report, nil); !strings.Contains(got[0], "Integrity verification: disabled") {
		t.Errorf("disabled verification: %v", got)
	}
	report.VerificationEnabled = true
	report.Files[1].Verified = true
	if got := uploadAlertDescriptions(report, nil); !strings.Contains(got[0], "Integrity verification: passed") {
		t.Errorf("passed verification: %v", got)
	}
}

func TestSendUploadAlert(t *testing.T) {
	var descriptions []string
	var titles []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Embeds []struct {
				Description string `json:"description"`
				Title       string `json:"title"`
			} `json:"embeds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.Embeds) != 1 {
			t.Errorf("invalid alert payload: %+v, %v", payload, err)
		} else {
			descriptions = append(descriptions, payload.Embeds[0].Description)
			titles = append(titles, payload.Embeds[0].Title)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	oldWebhook := *flags.DiscordWebHookUri
	t.Cleanup(func() { *flags.DiscordWebHookUri = oldWebhook })
	*flags.DiscordWebHookUri = server.URL
	alerting.Setup()
	user := users.User{UserID: "123", Gamertag: "Player"}
	sendUploadAlert(context.Background(), user, upload.Report{}, nil)
	if len(descriptions) != 0 {
		t.Fatal("empty successful upload sent an alert")
	}
	for _, verify := range []bool{false, true} {
		sendUploadAlert(context.Background(), user, upload.Report{
			VerificationEnabled: verify,
			Files: []upload.UploadedMedia{{
				Media: &models.Media{Category: models.MediaCategoryGameClip},
				URL:   "https://example.com/clip", Bytes: 10, Verified: verify,
			}},
		}, nil)
	}
	sendUploadAlert(context.Background(), user, upload.Report{VerificationEnabled: true}, errors.New("upload failed"))
	if len(descriptions) != 3 {
		t.Fatalf("got %d alerts, want 3", len(descriptions))
	}
	for i, want := range []string{"Integrity verification: disabled", "Integrity verification: passed", "Integrity verification: not performed"} {
		if titles[i] != "Xbox media upload: Player (123)" {
			t.Errorf("alert title = %q", titles[i])
		}
		if !strings.Contains(descriptions[i], want) {
			t.Errorf("alert %d missing %q: %s", i, want, descriptions[i])
		}
	}
	for _, description := range descriptions[:2] {
		if !strings.Contains(description, "https://example.com/clip") || !strings.Contains(description, "game clips: 1") || !strings.Contains(description, "Total bytes: 10") {
			t.Errorf("incomplete alert: %s", description)
		}
	}
}

func TestDoWorkConcurrentAndIndependent(t *testing.T) {
	firstErr, secondErr := errors.New("first upload"), errors.New("second upload")
	started := make(chan string, 3)
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- doWork(users.ApiKeyToUserMap{
			"key-a": {UserID: "a"}, "key-b": {UserID: "b"}, "key-c": {UserID: "c"},
			"key-a-duplicate": {UserID: "a"},
		}, func(keys []string, user users.User) error {
			if user.UserID == "a" && len(keys) != 2 {
				t.Errorf("user a has %d API keys, want 2", len(keys))
			}
			started <- user.UserID
			<-release
			switch user.UserID {
			case "a":
				return firstErr
			case "b":
				return secondErr
			default:
				return nil
			}
		})
	}()
	defer func() {
		if release != nil {
			close(release)
		}
	}()
	seen := map[string]bool{}
	for range 3 {
		select {
		case id := <-started:
			if seen[id] {
				t.Fatalf("user %q started more than once", id)
			}
			seen[id] = true
		case <-time.After(5 * time.Second):
			t.Fatal("users did not start concurrently")
		}
	}
	close(release)
	select {
	case err := <-done:
		release = nil
		if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
			t.Fatalf("missing user errors: %v", err)
		}
		if !strings.Contains(err.Error(), `user "a"`) || !strings.Contains(err.Error(), `user "b"`) {
			t.Fatalf("missing user context: %v", err)
		}
	case <-time.After(5 * time.Second):
		release = nil
		t.Fatal("work did not finish")
	}
	if err := doWork(nil, nil); err == nil {
		t.Fatal("empty users should fail")
	}
}

func TestUploadAlertDescriptionsSplit(t *testing.T) {
	report := upload.Report{}
	for i := 0; i < 100; i++ {
		report.Files = append(report.Files, upload.UploadedMedia{
			Media: &models.Media{}, URL: fmt.Sprintf("https://example.com/%03d/%s", i, strings.Repeat("x", 80)),
		})
	}
	got := uploadAlertDescriptions(report, nil)
	if len(got) < 2 {
		t.Fatal("expected multiple alert descriptions")
	}
	for _, description := range got {
		if len(description) > 3500 {
			t.Errorf("description too long: %d bytes", len(description))
		}
	}
	joined := strings.Join(got, "\n")
	for _, file := range report.Files {
		if strings.Count(joined, file.URL) != 1 {
			t.Errorf("URL not included exactly once: %s", file.URL)
		}
	}
}
