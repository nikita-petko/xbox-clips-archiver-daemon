package flags

import (
	"flag"
	"fmt"
	"os"
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
			glog.Infof("Flags setup complete!")
		}
	})
}
