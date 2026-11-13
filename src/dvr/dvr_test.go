package dvr

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	apphttp "github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/http"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/xbl/models"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetDVRMediaCategories(t *testing.T) {
	client := apphttp.Client().HTTPClient
	old := client.Transport
	t.Cleanup(func() { client.Transport = old })
	client.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.Path {
		case screenshotsEndpoint:
			body = `{"content":{"screenshots":[{"screenshotId":"s"}]}}`
		case gameclipsEndpoint:
			body = `{"content":{"gameClips":[{"gameClipId":"g"}]}}`
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	media, err := GetDVRMedia("key")
	if err != nil || len(media) != 2 {
		t.Fatalf("media = %v, error = %v", media, err)
	}
	if media[0].Id != "s" || media[0].Category != models.MediaCategoryScreenshot ||
		media[1].Id != "g" || media[1].Category != models.MediaCategoryGameClip {
		t.Fatalf("unexpected categories: %+v, %+v", media[0], media[1])
	}
}

func TestParseCaptureTime(t *testing.T) {
	for _, input := range []string{
		"2024-06-07 08:09:10Z",
		"2024-06-07T08:09:10Z",
		"2024-06-07T08:09:10.123Z",
	} {
		got, err := parseCaptureTime(input)
		if err != nil || got.UTC().Format("2006-01-02T15-04-05-0700") != "2024-06-07T08-09-10+0000" {
			t.Errorf("parse %q = %v, %v", input, got, err)
		}
	}
	got, err := parseCaptureTime("2024-06-07T10:09:10+02:00")
	if err != nil || !got.Equal(time.Date(2024, 6, 7, 8, 9, 10, 0, time.UTC)) {
		t.Errorf("offset parse = %v, %v", got, err)
	}
	if _, err := parseCaptureTime("invalid"); err == nil {
		t.Error("expected invalid timestamp error")
	}
}
