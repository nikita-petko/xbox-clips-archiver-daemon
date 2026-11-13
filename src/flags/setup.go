package flags

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/golang/glog"
)

var gSetupFlagsOnce sync.Once

// SetupFlags sets up flags for the current environment.
func SetupFlags(applicationName, buildMode, commitSha string) {
	gSetupFlagsOnce.Do(func() {
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s\nBuild Mode: %s\nCommit: %s %s\n\n", applicationName, buildMode, commitSha, FlagsUsageString)
			flag.PrintDefaults()
		}

		flag.Set("logtostderr", "true")
		flag.Set("v", "50")

		flag.Parse()

		applyEnvironmentVariableFlags()

		if !*HelpFlag {
			if err := ValidateUploaderGrouping(); err != nil {
				panic(err)
			}
			if err := ValidateUploadMediaFileNameMode(); err != nil {
				panic(err)
			}
			if _, err := ParseAlertURLBase(); err != nil {
				panic(err)
			}
			glog.Infof("Flags setup complete!")
		}

	})
}

// ParseAlertURLBase validates the optional base URL used for upload alerts.
func ParseAlertURLBase() (*url.URL, error) {
	if *AlertURLBase == "" {
		return nil, nil
	}
	base, err := url.Parse(*AlertURLBase)
	if err != nil {
		return nil, fmt.Errorf("invalid alert URL base: %w", err)
	}
	if base.Scheme != "https" || base.Hostname() == "" || base.User != nil ||
		base.Opaque != "" || base.RawQuery != "" || base.ForceQuery || strings.Contains(*AlertURLBase, "#") {
		return nil, fmt.Errorf("invalid alert URL base: expected an HTTPS URL without credentials, query, or fragment")
	}
	return base, nil
}

// ValidateUploadMediaFileNameMode checks the configured media file naming mode.
func ValidateUploadMediaFileNameMode() error {
	switch *UploadMediaFileNameMode {
	case "id", "captured":
		return nil
	default:
		return fmt.Errorf("invalid upload media file name mode %q: expected id or captured", *UploadMediaFileNameMode)
	}
}

// ValidateUploaderGrouping checks the configured upload grouping modes.
func ValidateUploaderGrouping() error {
	switch *UploaderGroupByUserMode {
	case "none", "id", "gamertag":
	default:
		return fmt.Errorf("invalid uploader group by user mode %q: expected none, id, or gamertag", *UploaderGroupByUserMode)
	}
	switch *UploaderGroupByTitleMode {
	case "none", "id", "name":
	default:
		return fmt.Errorf("invalid uploader group by title mode %q: expected none, id, or name", *UploaderGroupByTitleMode)
	}
	return nil
}
