package cache

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

func TestConcurrentUserDownloads(t *testing.T) {
	oldPath := *flags.CachePath
	t.Cleanup(func() { *flags.CachePath = oldPath })
	*flags.CachePath = t.TempDir()
	Setup()
	t.Cleanup(Close)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("media"))
	}))
	defer server.Close()

	const users = 40
	var wg sync.WaitGroup
	for i := range users {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := CacheGuardedHttpGet(fmt.Sprint(i), "title", "media", server.URL+"/file.mp4")
			if err != nil {
				t.Errorf("download user %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	data, err := os.ReadFile(filepath.Join(*flags.CachePath, downloadCacheFileName))
	if err != nil {
		t.Fatal(err)
	}
	var entries map[string]string
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != users {
		t.Fatalf("cached %d users, want %d", len(entries), users)
	}
}
