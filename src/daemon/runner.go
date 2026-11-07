package daemon

import (
	"time"

	"github.com/golang/glog"
	"github.vmminfra.dev/mfdlabs/xbox-clips-archiver-daemon/flags"
)

// Run runs the doWork func in a loop as well
// as initializes the context.
// Can only occur once.
func Run() {
	glog.Infof("Start work with %s interval.", *flags.Interval)

	for {
		err := DoWork()
		if err != nil {
			glog.Errorf("Error during work: %v", err)
		}

		time.Sleep(*flags.Interval)
	}
}
