package users

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
	client "github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/http"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchUsersConcurrently(t *testing.T) {
	oldPath := *flags.CachePath
	t.Cleanup(func() { *flags.CachePath = oldPath })
	*flags.CachePath = t.TempDir()
	cache.Setup()
	t.Cleanup(cache.Close)

	started := make(chan string, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	oldTransport := client.Client().HTTPClient.Transport
	t.Cleanup(func() { client.Client().HTTPClient.Transport = oldTransport })
	client.Client().HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		key := req.Header.Get("x-authorization")
		started <- key
		<-release
		body := fmt.Sprintf(`{"content":{"profileUsers":[{"id":%q,"settings":[{"id":"Gamertag","value":%q}]}]}}`, key, "Player "+key)
		if key == "secret-bad-key" {
			body = `{"content":{"profileUsers":[]}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})

	type result struct {
		users ApiKeyToUserMap
		err   error
	}
	done := make(chan result, 1)
	go func() {
		profiles, err := fetchUsersByUserIds([]string{"a", "b", "a"})
		done <- result{users: profiles, err: err}
	}()
	seen := map[string]bool{}
	for range 2 {
		select {
		case key := <-started:
			seen[key] = true
		case <-time.After(5 * time.Second):
			t.Fatal("profile requests did not start concurrently")
		}
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("unexpected requests: %v", seen)
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case got := <-done:
		if got.err != nil || len(got.users) != 2 || got.users["a"].Gamertag != "Player a" || got.users["b"].UserID != "b" {
			t.Fatalf("profiles = %v, error = %v", got.users, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("profile requests did not finish")
	}
	profiles, err := fetchUsersByUserIds([]string{"secret-bad-key"})
	if err == nil || profiles != nil || !strings.Contains(err.Error(), "API key #1") || strings.Contains(err.Error(), "secret-bad-key") {
		t.Fatalf("profiles = %v, error = %v", profiles, err)
	}
}
