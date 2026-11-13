package s3

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
	"testing"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/cache"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

func TestValidateIntegrity(t *testing.T) {
	resetSetup(t)
	if _, err := ValidateIntegrity(context.Background(), "key", nil); err == nil {
		t.Fatal("expected error before setup")
	}

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "123456789")
	}))
	defer source.Close()
	oldCachePath := *flags.CachePath
	*flags.CachePath = t.TempDir()
	t.Cleanup(func() { *flags.CachePath = oldCachePath })
	cache.Setup()
	t.Cleanup(cache.Close)
	guard, err := cache.CacheGuardedHttpGet("user", "title", "media", source.URL+"/file")
	if err != nil {
		t.Fatal(err)
	}

	first := sha256.Sum256([]byte("1234"))
	second := sha256.Sum256([]byte("56789"))
	joined := append(first[:], second[:]...)
	sum := sha256.Sum256(joined)
	composite := base64.StdEncoding.EncodeToString(sum[:]) + "-2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			http.Error(w, "unexpected method", http.StatusBadRequest)
			return
		}
		key := strings.TrimPrefix(r.URL.Path, "/clips/")
		if part := r.URL.Query().Get("partNumber"); part != "" {
			if key == "part-forbidden" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if r.Header.Get("If-Match") != `"object"` || r.URL.Query().Get("versionId") != "version" {
				w.WriteHeader(http.StatusPreconditionFailed)
				return
			}
			switch part {
			case "1":
				w.Header().Set("Content-Length", "4")
			case "2":
				w.Header().Set("Content-Length", "5")
			default:
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}
		if r.Header.Get("X-Amz-Checksum-Mode") != "ENABLED" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch key {
		case "sha256":
			w.Header().Set("x-amz-checksum-sha256", "FeKw08M4keuw8e9gnsQZQgwg4yDOlMZfvIwzEkSOsiU=")
		case "sha1":
			w.Header().Set("x-amz-checksum-sha1", "98O8HYCOBHMq32eZZczDTKeuNEE=")
		case "crc32":
			w.Header().Set("x-amz-checksum-crc32", "y/Q5Jg==")
		case "crc32c":
			w.Header().Set("x-amz-checksum-crc32c", "4waSgw==")
		case "crc64":
			w.Header().Set("x-amz-checksum-crc64nvme", "rosUhgp5mIg=")
		case "mismatch":
			w.Header().Set("x-amz-checksum-sha256", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
		case "composite", "composite-mismatch", "legacy-composite", "short-file", "long-file", "part-forbidden":
			w.Header().Set("x-amz-checksum-sha256", composite)
			if key != "legacy-composite" {
				w.Header().Set("x-amz-checksum-type", "COMPOSITE")
			}
			w.Header().Set("Etag", `"object"`)
			w.Header().Set("x-amz-version-id", "version")
		case "invalid-base64":
			w.Header().Set("x-amz-checksum-sha256", "invalid!")
		case "invalid-length":
			w.Header().Set("x-amz-checksum-sha256", "AAAA")
		case "invalid-count":
			w.Header().Set("x-amz-checksum-sha256", composite+"-3")
		case "no-count":
			w.Header().Set("x-amz-checksum-sha256", "FeKw08M4keuw8e9gnsQZQgwg4yDOlMZfvIwzEkSOsiU=")
			w.Header().Set("x-amz-checksum-type", "COMPOSITE")
		case "unexpected-count":
			w.Header().Set("x-amz-checksum-sha256", composite)
			w.Header().Set("x-amz-checksum-type", "FULL_OBJECT")
		case "unknown-type":
			w.Header().Set("x-amz-checksum-sha256", composite)
			w.Header().Set("x-amz-checksum-type", "UNKNOWN")
		case "no-identity":
			w.Header().Set("x-amz-checksum-sha256", composite)
		case "no-checksum":
		case "forbidden":
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	t.Setenv("AWS_ENDPOINT_URL_S3", server.URL)
	*flags.S3Bucket, *flags.S3Region = "clips", "us-east-1"
	*flags.S3AccessKey, *flags.S3SecretAccessKey = "access", "secret"
	Setup()

	for _, tc := range []struct {
		key       string
		data      string
		want      bool
		wantError bool
	}{
		{"sha256", "123456789", true, false},
		{"sha1", "123456789", true, false},
		{"crc32", "123456789", true, false},
		{"crc32c", "123456789", true, false},
		{"crc64", "123456789", true, false},
		{"mismatch", "123456789", false, false},
		{"composite", "123456789", true, false},
		{"composite-mismatch", "123456780", false, false},
		{"legacy-composite", "123456789", true, false},
		{"short-file", "12345678", false, false},
		{"long-file", "1234567890", false, false},
		{"invalid-base64", "123456789", false, true},
		{"invalid-length", "123456789", false, true},
		{"invalid-count", "123456789", false, true},
		{"no-count", "123456789", false, true},
		{"unexpected-count", "123456789", false, true},
		{"unknown-type", "123456789", false, true},
		{"no-identity", "123456789", false, true},
		{"no-checksum", "123456789", false, true},
		{"forbidden", "123456789", false, true},
		{"part-forbidden", "123456789", false, true},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if err := os.WriteFile(guard.GetFilePath(), []byte(tc.data), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := ValidateIntegrity(context.Background(), tc.key, guard)
			if got != tc.want || (err != nil) != tc.wantError {
				t.Fatalf("got (%v, %v), want (%v, error=%v)", got, err, tc.want, tc.wantError)
			}
		})
	}
	if _, err := ValidateIntegrity(context.Background(), "", guard); err == nil {
		t.Error("expected empty key error")
	}
	if _, err := ValidateIntegrity(context.Background(), "key", nil); err == nil {
		t.Error("expected nil guard error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ValidateIntegrity(ctx, "sha256", guard); !errors.Is(err, context.Canceled) {
		t.Errorf("expected canceled context, got %v", err)
	}
	if err := os.Remove(guard.GetFilePath()); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateIntegrity(context.Background(), "sha256", guard); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected missing file error, got %v", err)
	}
}

func TestContextReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := io.ReadAll(&contextReader{ctx: ctx, reader: strings.NewReader("test")})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(fmt.Errorf("expected canceled read: %w", err))
	}
}
