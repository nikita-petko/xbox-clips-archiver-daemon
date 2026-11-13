package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

func TestUploadAndExists(t *testing.T) {
	const contents = "a video clip"
	resetSetup(t)
	if err := Upload(context.Background(), "key", nil); err == nil {
		t.Fatal("expected error before setup")
	}
	if _, err := Exists(context.Background(), "key"); err == nil {
		t.Fatal("expected error before setup")
	}
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, contents)
	}))
	defer source.Close()

	oldCachePath := *flags.CachePath
	*flags.CachePath = t.TempDir()
	t.Cleanup(func() { *flags.CachePath = oldCachePath })
	cache.Setup()
	t.Cleanup(cache.Close)
	guard, err := cache.CacheGuardedHttpGet("user", "title", "media", source.URL+"/video.mp4")
	if err != nil {
		t.Fatal(err)
	}

	var uploaded, missingBucket atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Authorization"), "Credential=access/") ||
			!strings.Contains(r.Header.Get("Authorization"), "/us-west-2/s3/") ||
			r.Header.Get("X-Amz-Security-Token") != "session-token" {
			http.Error(w, "unexpected credentials or region", http.StatusBadRequest)
			return
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/clips/forbidden":
			w.WriteHeader(http.StatusForbidden)
		case r.Method == http.MethodPut && r.URL.Path == "/clips/folder/video.mp4":
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != contents {
				http.Error(w, "unexpected body", http.StatusBadRequest)
				return
			}
			uploaded.Store(true)
		case r.Method == http.MethodHead && r.URL.Path == "/clips/folder/video.mp4":
			if !uploaded.Load() {
				w.WriteHeader(http.StatusNotFound)
			}
		case r.Method == http.MethodHead && r.URL.Path == "/clips/missing":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodHead && r.URL.Path == "/clips/forbidden":
			w.WriteHeader(http.StatusForbidden)
		case r.Method == http.MethodHead && strings.TrimSuffix(r.URL.Path, "/") == "/clips":
			if missingBucket.Load() {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			http.Error(w, fmt.Sprintf("unexpected request %s %s", r.Method, r.URL.Path), http.StatusBadRequest)
		}
	}))
	defer server.Close()

	t.Setenv("AWS_ENDPOINT_URL_S3", server.URL)
	t.Setenv("AWS_SESSION_TOKEN", "session-token")
	*flags.S3Bucket = "clips"
	*flags.S3Region = "us-west-2"
	*flags.S3AccessKey = "access"
	*flags.S3SecretAccessKey = "secret"
	Setup()
	*flags.S3Bucket = "other"
	Setup()

	exists, err := Exists(context.Background(), "folder/video.mp4")
	if err != nil || exists {
		t.Fatalf("before upload: exists = %v, error = %v", exists, err)
	}
	if err := Upload(context.Background(), "folder/video.mp4", guard); err != nil {
		t.Fatalf("upload: %v", err)
	}
	exists, err = Exists(context.Background(), "folder/video.mp4")
	if err != nil || !exists {
		t.Fatalf("after upload: exists = %v, error = %v", exists, err)
	}
	exists, err = Exists(context.Background(), "missing")
	if err != nil || exists {
		t.Fatalf("missing key: exists = %v, error = %v", exists, err)
	}
	if _, err := Exists(context.Background(), "forbidden"); err == nil {
		t.Error("expected forbidden error")
	}
	if err := Upload(context.Background(), "forbidden", guard); err == nil {
		t.Error("expected forbidden upload error")
	}
	missingBucket.Store(true)
	if _, err := Exists(context.Background(), "missing"); err == nil {
		t.Error("expected missing bucket error")
	}
	missingBucket.Store(false)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Upload(ctx, "folder/video.mp4", guard); !errors.Is(err, context.Canceled) {
		t.Errorf("expected canceled upload, got %v", err)
	}
	if _, err := Exists(ctx, "folder/video.mp4"); !errors.Is(err, context.Canceled) {
		t.Errorf("expected canceled check, got %v", err)
	}

	if err := os.Remove(guard.GetFilePath()); err != nil {
		t.Fatal(err)
	}
	if err := Upload(context.Background(), "folder/video.mp4", guard); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected missing file error, got %v", err)
	}
}

func TestConfigurationAndInvalidInputs(t *testing.T) {
	resetSetup(t)
	*flags.S3Bucket = "clips"
	*flags.S3Region = "us-east-1"
	*flags.S3AccessKey = ""
	*flags.S3SecretAccessKey = ""
	t.Setenv("AWS_ACCESS_KEY_ID", "chain-access")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "chain-secret")
	Setup()
	credentials, err := gS3Client.Options().Credentials.Retrieve(context.Background())
	if err != nil || credentials.AccessKeyID != "chain-access" {
		t.Fatalf("expected default credential chain, got %v, %v", credentials.Source, err)
	}
	if err := Upload(context.Background(), "key", nil); err == nil {
		t.Error("expected nil guard error")
	}
	if err := Upload(context.Background(), "", &cache.CacheGuard{}); err == nil {
		t.Error("expected empty upload key error")
	}
	if _, err := Exists(context.Background(), ""); err == nil {
		t.Error("expected empty lookup key error")
	}
	if err := Upload(context.Background(), "key", &cache.CacheGuard{}); err == nil || !strings.Contains(err.Error(), "open cached file") {
		t.Errorf("expected file open error, got %v", err)
	}
}

func TestSetupInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, bucket, region, access, secret string
	}{
		{"missing bucket", "", "us-east-1", "", ""},
		{"missing region", "clips", "", "", ""},
		{"missing secret", "clips", "us-east-1", "access", ""},
		{"missing access", "clips", "us-east-1", "", "secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetSetup(t)
			*flags.S3Bucket, *flags.S3Region = tc.bucket, tc.region
			*flags.S3AccessKey, *flags.S3SecretAccessKey = tc.access, tc.secret
			defer func() {
				if recover() == nil {
					t.Error("expected setup to panic")
				}
			}()
			Setup()
		})
	}
}

func TestSetupConfigLoadFailure(t *testing.T) {
	resetSetup(t)
	*flags.S3Bucket, *flags.S3Region = "clips", "us-east-1"
	*flags.S3AccessKey, *flags.S3SecretAccessKey = "", ""
	t.Setenv("AWS_PROFILE", "nonexistent-profile")
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()+"/missing-config")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", t.TempDir()+"/missing-credentials")
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "Error initializing AWS S3 config") {
			t.Errorf("expected config loading panic, got %v", r)
		}
	}()
	Setup()
}

func resetSetup(t *testing.T) {
	t.Helper()
	bucket, region := *flags.S3Bucket, *flags.S3Region
	access, secret := *flags.S3AccessKey, *flags.S3SecretAccessKey
	gS3Client, gS3Uploader, gS3Bucket = nil, nil, ""
	gSetupS3Once = sync.Once{}
	t.Cleanup(func() {
		*flags.S3Bucket, *flags.S3Region = bucket, region
		*flags.S3AccessKey, *flags.S3SecretAccessKey = access, secret
		gS3Client, gS3Uploader, gS3Bucket = nil, nil, ""
		gSetupS3Once = sync.Once{}
	})
}
