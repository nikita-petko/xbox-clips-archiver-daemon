package daemon

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/alerting"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	client "github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/http"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/s3"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/upload"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/users"
)

type xblTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t xblTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.xbl.io" {
		req = req.Clone(req.Context())
		req.URL.Scheme, req.URL.Host = t.target.Scheme, t.target.Host
		req.Host = t.target.Host
	}
	return t.base.RoundTrip(req)
}

func TestUserPipelinesAndAlerts(t *testing.T) {
	type embed struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	alerts := make(chan embed, 2)
	firstStarted := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	var requestsMu sync.Mutex
	requests := make(map[string]int)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/dvr/screenshots":
			id := r.Header.Get("x-authorization")
			if id == "0bad" {
				_, _ = w.Write([]byte(`{"content":`))
				return
			}
			if id == "a" {
				close(firstStarted)
				<-release
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"content": map[string]any{
				"screenshots": []any{map[string]any{
					"screenshotId": id, "titleId": 1,
					"screenshotUris": []any{map[string]string{"uri": server.URL + "/media/" + id + ".jpg"}},
				}},
			}})
		case r.URL.Path == "/v2/dvr/gameclips":
			_, _ = w.Write([]byte(`{"content":{"gameClips":[]}}`))
		case strings.HasPrefix(r.URL.Path, "/media/"):
			_, _ = w.Write([]byte(r.URL.Path))
		case r.Method == http.MethodHead && (r.URL.Path == "/clips/a/1/a.jpg" || r.URL.Path == "/clips/b/1/b.jpg"):
			requestsMu.Lock()
			requests[r.Method+" "+r.URL.Path]++
			requestsMu.Unlock()
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodHead && strings.TrimSuffix(r.URL.Path, "/") == "/clips":
			requestsMu.Lock()
			requests["HEAD /clips"]++
			requestsMu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && (r.URL.Path == "/clips/a/1/a.jpg" || r.URL.Path == "/clips/b/1/b.jpg"):
			body, err := io.ReadAll(r.Body)
			userID := strings.Split(r.URL.Path, "/")[2]
			if err != nil || string(body) != "/media/"+userID+".jpg" || r.Header.Get("Content-Type") != "image/jpeg" {
				http.Error(w, "unexpected upload body or content type", http.StatusBadRequest)
				return
			}
			requestsMu.Lock()
			requests[r.Method+" "+r.URL.Path]++
			requestsMu.Unlock()
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/webhook":
			var body struct {
				Embeds []embed `json:"embeds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Embeds) != 1 {
				t.Errorf("invalid alert: %+v, %v", body, err)
			} else {
				alerts <- body.Embeds[0]
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	defer server.Close()
	defer releaseOnce.Do(func() { close(release) })

	oldCache, oldWebhook := *flags.CachePath, *flags.DiscordWebHookUri
	oldBucket, oldRegion, oldAccess, oldSecret := *flags.S3Bucket, *flags.S3Region, *flags.S3AccessKey, *flags.S3SecretAccessKey
	oldUser, oldTitle, oldFile := *flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode, *flags.UploadMediaFileNameMode
	oldVerify, oldPrefix, oldAlertBase := *flags.ShouldVerifyUploadIntegrity, *flags.S3PathPrefix, *flags.AlertURLBase
	t.Cleanup(func() {
		*flags.CachePath, *flags.DiscordWebHookUri = oldCache, oldWebhook
		*flags.S3Bucket, *flags.S3Region, *flags.S3AccessKey, *flags.S3SecretAccessKey = oldBucket, oldRegion, oldAccess, oldSecret
		*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode, *flags.UploadMediaFileNameMode = oldUser, oldTitle, oldFile
		*flags.ShouldVerifyUploadIntegrity, *flags.S3PathPrefix, *flags.AlertURLBase = oldVerify, oldPrefix, oldAlertBase
	})
	*flags.CachePath, *flags.DiscordWebHookUri = t.TempDir(), server.URL+"/webhook"
	*flags.S3Bucket, *flags.S3Region, *flags.S3AccessKey, *flags.S3SecretAccessKey = "clips", "us-east-1", "access", "secret"
	*flags.UploaderGroupByUserMode, *flags.UploaderGroupByTitleMode, *flags.UploadMediaFileNameMode = "id", "id", "id"
	*flags.ShouldVerifyUploadIntegrity, *flags.S3PathPrefix, *flags.AlertURLBase = false, "", ""
	t.Setenv("AWS_ENDPOINT_URL_S3", server.URL)
	cache.Setup()
	t.Cleanup(cache.Close)
	s3.Setup()
	alerting.Setup()
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	oldTransport := client.Client().HTTPClient.Transport
	client.Client().HTTPClient.Transport = xblTransport{target: target, base: http.DefaultTransport}
	t.Cleanup(func() { client.Client().HTTPClient.Transport = oldTransport })

	done := make(chan error, 1)
	go func() {
		session := new(upload.Session)
		done <- doWork(users.ApiKeyToUserMap{
			"0bad": {UserID: "a", Gamertag: "Alice"},
			"a":    {UserID: "a", Gamertag: "Alice"},
			"b":    {UserID: "b", Gamertag: "Bob"},
		}, func(keys []string, user users.User) error {
			return doUserWork(context.Background(), session, keys, user)
		})
	}()
	select {
	case <-firstStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("first user did not start")
	}
	var bob embed
	select {
	case bob = <-alerts:
	case <-time.After(5 * time.Second):
		t.Fatal("second user's alert waited for the first user's download")
	}
	if bob.Title != "Xbox media upload: Bob (b)" || !strings.Contains(bob.Description, "/b/1/b.jpg") || strings.Contains(bob.Description, "/a/1/a.jpg") {
		t.Errorf("incorrect Bob alert: %+v", bob)
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("user pipelines did not finish")
	}
	select {
	case alice := <-alerts:
		if alice.Title != "Xbox media upload: Alice (a)" || !strings.Contains(alice.Description, "/a/1/a.jpg") || strings.Contains(alice.Description, "/b/1/b.jpg") {
			t.Errorf("incorrect Alice alert: %+v", alice)
		}
	default:
		t.Fatal("missing Alice alert")
	}
	requestsMu.Lock()
	defer requestsMu.Unlock()
	for request, want := range map[string]int{
		"HEAD /clips/a/1/a.jpg": 1,
		"HEAD /clips/b/1/b.jpg": 1,
		"HEAD /clips":           2,
		"PUT /clips/a/1/a.jpg":  1,
		"PUT /clips/b/1/b.jpg":  1,
	} {
		if got := requests[request]; got != want {
			t.Errorf("%s: got %d requests, want %d", request, got, want)
		}
	}
}
