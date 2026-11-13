package upload

import (
	"context"
	"strings"
	"testing"

	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

func TestAlertURLOverride(t *testing.T) {
	old := *flags.AlertURLBase
	t.Cleanup(func() { *flags.AlertURLBase = old })
	for _, tc := range []struct {
		base, key, want string
	}{
		{
			"https://share.cdn.vmminfra.dev",
			"xbox/Tom Clancy’s The Division/2026-10-15T07-14-44+0000.MP4",
			"https://share.cdn.vmminfra.dev/xbox/Tom%20Clancy%E2%80%99s%20The%20Division/2026-10-15T07-14-44%2B0000.MP4",
		},
		{"https://cdn.example.com", "x/a+b c/%2B.mp4", "https://cdn.example.com/x/a%2Bb%20c/%252B.mp4"},
		{"https://cdn.example.com/assets/", "xbox/a%2Fb #?.mp4", "https://cdn.example.com/assets/xbox/a%252Fb%20%23%3F.mp4"},
		{"https://cdn.example.com/a%2Fb/", "x", "https://cdn.example.com/a%2Fb/x"},
		{"https://cdn.example.com/a%2F", "x", "https://cdn.example.com/a%2F/x"},
		{"https://cdn.example.com", "/x", "https://cdn.example.com//x"},
	} {
		*flags.AlertURLBase = tc.base
		got, err := alertURL(context.Background(), tc.key)
		if err != nil || got != tc.want {
			t.Errorf("alertURL(%q, %q) = %q, %v; want %q", tc.base, tc.key, got, err, tc.want)
		}
	}
	*flags.AlertURLBase = "https://cdn.example.com"
	if _, err := alertURL(context.Background(), ""); err == nil {
		t.Error("empty key should fail")
	}
	*flags.AlertURLBase = "http://cdn.example.com"
	if _, err := alertURL(context.Background(), "key"); err == nil || !strings.Contains(err.Error(), "invalid alert URL base") {
		t.Errorf("invalid base: %v", err)
	}
}
